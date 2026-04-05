package bridge

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/pion/ice/v4"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const (
	desktopWebRTCPeerTimeout       = 45 * time.Second
	desktopWebRTCOfferWaitTimeout  = 8 * time.Second
	desktopWebRTCAnswerWaitTimeout = 8 * time.Second
)

type desktopWebRTCService struct {
	logger *log.Logger
	api    *webrtc.API
}

type desktopWebRTCH264Format struct {
	frameRate           int
	nalUnitHeaderLength int
	sps                 []byte
	pps                 []byte
}

type desktopWebRTCSample struct {
	ptsUs      int64
	durationUs int64
	keyFrame   bool
	data       []byte
}

type desktopWebRTCPeer struct {
	id         string
	logger     *log.Logger
	session    *desktopSession
	pc         *webrtc.PeerConnection
	videoTrack *webrtc.TrackLocalStaticSample
	rtpSender  *webrtc.RTPSender

	mu           sync.Mutex
	streaming    bool
	subscriberID int64
	closeOnce    sync.Once
}

func newDesktopWebRTCService(logger *log.Logger, port int) (*desktopWebRTCService, error) {
	if port <= 0 {
		port = 8787
	}

	var mediaEngine webrtc.MediaEngine
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		return nil, fmt.Errorf("failed to register WebRTC codecs: %w", err)
	}

	interceptorRegistry := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(&mediaEngine, interceptorRegistry); err != nil {
		return nil, fmt.Errorf("failed to register WebRTC interceptors: %w", err)
	}

	settingEngine := webrtc.SettingEngine{}
	udpMux, err := ice.NewMultiUDPMuxFromPort(port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen for WebRTC UDP traffic on %d: %w", port, err)
	}
	settingEngine.SetICEUDPMux(udpMux)

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(&mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
		webrtc.WithSettingEngine(settingEngine),
	)
	return &desktopWebRTCService{
		logger: logger,
		api:    api,
	}, nil
}

func (m *desktopSessionManager) webRTCService() (*desktopWebRTCService, error) {
	m.webrtcOnce.Do(func() {
		m.webrtc, m.webrtcError = newDesktopWebRTCService(m.logger, m.port)
	})
	return m.webrtc, m.webrtcError
}

func (m *desktopSessionManager) WebRTCOffer(params map[string]any) (map[string]any, error) {
	session, err := m.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return nil, err
	}
	service, err := m.webRTCService()
	if err != nil {
		return nil, err
	}
	return session.createWebRTCOffer(service)
}

func (m *desktopSessionManager) WebRTCAnswer(params map[string]any) (map[string]any, error) {
	session, err := m.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return nil, err
	}
	peerID, _ := stringParam(params, "peerId")
	if peerID == "" {
		return nil, errors.New("desktop webrtc answer is missing peerId")
	}
	sdp, _ := stringParam(params, "sdp")
	if sdp == "" {
		return nil, errors.New("desktop webrtc answer is missing sdp")
	}
	return session.applyWebRTCAnswer(peerID, sdp)
}

func (b *Bridge) desktopSessionWebRTCOffer(params map[string]any) map[string]any {
	if b.desktopSessions == nil {
		return map[string]any{"error": "Desktop sessions are unavailable on this bridge."}
	}
	result, err := b.desktopSessions.WebRTCOffer(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) desktopSessionWebRTCAnswer(params map[string]any) map[string]any {
	if b.desktopSessions == nil {
		return map[string]any{"error": "Desktop sessions are unavailable on this bridge."}
	}
	result, err := b.desktopSessions.WebRTCAnswer(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (s *desktopSession) createWebRTCOffer(service *desktopWebRTCService) (map[string]any, error) {
	if service == nil {
		return nil, errors.New("desktop WebRTC service is unavailable")
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errors.New("desktop session closed")
	}
	video := s.video
	s.mu.Unlock()
	if video == nil {
		return nil, errors.New("desktop video stream is unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), desktopWebRTCOfferWaitTimeout)
	defer cancel()
	if err := video.waitReady(ctx, desktopWebRTCOfferWaitTimeout); err != nil {
		return nil, err
	}

	peer, err := newDesktopWebRTCPeer(service, s)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	if s.webrtcPeers == nil {
		s.webrtcPeers = map[string]*desktopWebRTCPeer{}
	}
	s.webrtcPeers[peer.id] = peer
	s.mu.Unlock()

	time.AfterFunc(desktopWebRTCPeerTimeout, func() {
		peer.mu.Lock()
		streaming := peer.streaming
		peer.mu.Unlock()
		if !streaming {
			peer.close()
		}
	})

	offer, err := peer.createOffer()
	if err != nil {
		s.removeWebRTCPeer(peer.id)
		peer.close()
		return nil, err
	}

	return map[string]any{
		"peerId": peer.id,
		"type":   "offer",
		"sdp":    offer.SDP,
	}, nil
}

func (s *desktopSession) applyWebRTCAnswer(peerID, sdp string) (map[string]any, error) {
	peer := s.webRTCPeer(peerID)
	if peer == nil {
		return nil, errors.New("desktop WebRTC peer was not found")
	}
	if err := peer.applyAnswer(sdp); err != nil {
		s.removeWebRTCPeer(peerID)
		peer.close()
		return nil, err
	}
	return s.statusMap(), nil
}

func (s *desktopSession) webRTCPeer(peerID string) *desktopWebRTCPeer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.webrtcPeers[strings.TrimSpace(peerID)]
}

func (s *desktopSession) removeWebRTCPeer(peerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.webrtcPeers, strings.TrimSpace(peerID))
}

func newDesktopWebRTCPeer(service *desktopWebRTCService, session *desktopSession) (*desktopWebRTCPeer, error) {
	if service == nil || service.api == nil {
		return nil, errors.New("desktop WebRTC API is unavailable")
	}
	peerID, err := randomMobileResourceID()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate desktop WebRTC peer: %w", err)
	}

	pc, err := service.api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, fmt.Errorf("failed to create desktop WebRTC peer connection: %w", err)
	}

	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeH264,
		},
		"desktop-video",
		peerID,
	)
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("failed to create desktop WebRTC track: %w", err)
	}

	rtpSender, err := pc.AddTrack(videoTrack)
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("failed to add desktop WebRTC track: %w", err)
	}

	peer := &desktopWebRTCPeer{
		id:         peerID,
		logger:     service.logger,
		session:    session,
		pc:         pc,
		videoTrack: videoTrack,
		rtpSender:  rtpSender,
	}

	go peer.readRTCP()

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		service.logger.Printf("desktop webrtc peer %s state=%s", peerID, state.String())
		switch state {
		case webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:
			session.removeWebRTCPeer(peerID)
			peer.close()
		}
	})
	return peer, nil
}

func (p *desktopWebRTCPeer) createOffer() (webrtc.SessionDescription, error) {
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("failed to create desktop WebRTC offer: %w", err)
	}
	gatherComplete := webrtc.GatheringCompletePromise(p.pc)
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return webrtc.SessionDescription{}, fmt.Errorf("failed to set desktop WebRTC local description: %w", err)
	}

	select {
	case <-gatherComplete:
	case <-time.After(desktopWebRTCOfferWaitTimeout):
		return webrtc.SessionDescription{}, errors.New("desktop WebRTC offer timed out while gathering ICE candidates")
	}

	localDescription := p.pc.LocalDescription()
	if localDescription == nil {
		return webrtc.SessionDescription{}, errors.New("desktop WebRTC local description was unavailable")
	}
	return *localDescription, nil
}

func (p *desktopWebRTCPeer) applyAnswer(sdp string) error {
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}
	if err := p.pc.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set desktop WebRTC remote description: %w", err)
	}

	p.mu.Lock()
	alreadyStreaming := p.streaming
	if !p.streaming {
		p.streaming = true
	}
	p.mu.Unlock()
	if !alreadyStreaming {
		go p.forwardSamples()
	}
	return nil
}

func (p *desktopWebRTCPeer) forwardSamples() {
	subscriberID, subscriber, initialFormat, initialSample, err := p.session.subscribeVideoStream()
	if err != nil {
		p.logger.Printf("desktop webrtc peer %s failed to subscribe video stream: %v", p.id, err)
		p.session.removeWebRTCPeer(p.id)
		p.close()
		return
	}

	p.mu.Lock()
	p.subscriberID = subscriberID
	p.mu.Unlock()
	defer p.session.unsubscribeVideoStream(subscriberID)

	currentFormat, _ := parseDesktopWebRTCH264Format(initialFormat)
	if currentFormat != nil && len(initialSample) != 0 {
		if err := p.writeSamplePacket(initialSample, *currentFormat); err != nil && !errors.Is(err, io.ErrClosedPipe) {
			p.logger.Printf("desktop webrtc peer %s failed to write initial sample: %v", p.id, err)
		}
	}

	for event := range subscriber {
		switch event.Kind {
		case "json":
			if nextFormat, err := parseDesktopWebRTCH264Format(event.JSON); err == nil && nextFormat != nil {
				currentFormat = nextFormat
			}
		case "sample":
			if currentFormat == nil {
				continue
			}
			if err := p.writeSamplePacket(event.Sample, *currentFormat); err != nil {
				if !errors.Is(err, io.ErrClosedPipe) {
					p.logger.Printf("desktop webrtc peer %s failed to write sample: %v", p.id, err)
				}
				p.session.removeWebRTCPeer(p.id)
				p.close()
				return
			}
		}
	}
}

func (p *desktopWebRTCPeer) writeSamplePacket(packet []byte, format desktopWebRTCH264Format) error {
	sample, err := parseDesktopWebRTCSamplePacket(packet)
	if err != nil {
		return err
	}
	annexB, err := desktopWebRTCSampleToAnnexB(sample, format)
	if err != nil {
		return err
	}

	duration := time.Duration(sample.durationUs) * time.Microsecond
	if duration <= 0 {
		frameRate := maxInt(format.frameRate, 30)
		duration = time.Second / time.Duration(frameRate)
	}
	return p.videoTrack.WriteSample(media.Sample{
		Data:     annexB,
		Duration: duration,
	})
}

func (p *desktopWebRTCPeer) readRTCP() {
	rtcpBuf := make([]byte, 1500)
	for {
		if _, _, err := p.rtpSender.Read(rtcpBuf); err != nil {
			return
		}
	}
}

func (p *desktopWebRTCPeer) close() {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		subscriberID := p.subscriberID
		p.subscriberID = 0
		p.mu.Unlock()
		if subscriberID != 0 && p.session != nil {
			p.session.unsubscribeVideoStream(subscriberID)
		}
		if p.pc != nil {
			_ = p.pc.Close()
		}
	})
}

func parseDesktopWebRTCH264Format(message map[string]any) (*desktopWebRTCH264Format, error) {
	if message == nil || strings.TrimSpace(stringValueFromJSON(message, "type")) != "format" {
		return nil, nil
	}
	payload, _ := message["payload"].(map[string]any)
	if payload == nil {
		return nil, errors.New("desktop WebRTC format event did not include payload")
	}

	spsBase64 := strings.TrimSpace(stringValueFromJSON(payload, "spsBase64"))
	ppsBase64 := strings.TrimSpace(stringValueFromJSON(payload, "ppsBase64"))
	if spsBase64 == "" || ppsBase64 == "" {
		return nil, errors.New("desktop WebRTC format event was missing parameter sets")
	}

	sps, err := base64.StdEncoding.DecodeString(spsBase64)
	if err != nil {
		return nil, fmt.Errorf("desktop WebRTC format sps decode failed: %w", err)
	}
	pps, err := base64.StdEncoding.DecodeString(ppsBase64)
	if err != nil {
		return nil, fmt.Errorf("desktop WebRTC format pps decode failed: %w", err)
	}

	nalUnitHeaderLength := maxInt(int(float64Value(payload["nalUnitHeaderLength"])), 4)
	return &desktopWebRTCH264Format{
		frameRate:           maxInt(int(float64Value(payload["frameRate"])), 30),
		nalUnitHeaderLength: nalUnitHeaderLength,
		sps:                 sps,
		pps:                 pps,
	}, nil
}

func parseDesktopWebRTCSamplePacket(packet []byte) (desktopWebRTCSample, error) {
	if len(packet) < 17 {
		return desktopWebRTCSample{}, errors.New("desktop WebRTC sample packet was truncated")
	}
	sample := desktopWebRTCSample{
		ptsUs:      int64(binary.BigEndian.Uint64(packet[0:8])),
		durationUs: int64(binary.BigEndian.Uint64(packet[8:16])),
		keyFrame:   packet[16] == 1,
		data:       cloneBytes(packet[17:]),
	}
	if len(sample.data) == 0 {
		return desktopWebRTCSample{}, errors.New("desktop WebRTC sample packet was missing frame data")
	}
	return sample, nil
}

func desktopWebRTCSampleToAnnexB(sample desktopWebRTCSample, format desktopWebRTCH264Format) ([]byte, error) {
	headerLength := format.nalUnitHeaderLength
	if headerLength <= 0 || headerLength > 4 {
		return nil, fmt.Errorf("unsupported H264 NAL header length: %d", headerLength)
	}

	output := make([]byte, 0, len(sample.data)+128)
	if sample.keyFrame {
		output = appendAnnexBNAL(output, format.sps)
		output = appendAnnexBNAL(output, format.pps)
	}

	offset := 0
	for offset+headerLength <= len(sample.data) {
		nalSize := 0
		for i := 0; i < headerLength; i++ {
			nalSize = (nalSize << 8) | int(sample.data[offset+i])
		}
		offset += headerLength
		if nalSize <= 0 || offset+nalSize > len(sample.data) {
			return nil, errors.New("desktop WebRTC sample contained an invalid NAL length")
		}
		output = appendAnnexBNAL(output, sample.data[offset:offset+nalSize])
		offset += nalSize
	}
	if len(output) == 0 {
		return nil, errors.New("desktop WebRTC sample did not contain any NAL units")
	}
	return output, nil
}

func appendAnnexBNAL(dst []byte, nal []byte) []byte {
	if len(nal) == 0 {
		return dst
	}
	dst = append(dst, 0x00, 0x00, 0x00, 0x01)
	return append(dst, nal...)
}

func float64Value(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}
