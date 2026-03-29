package com.boomyao.codexmobile.tailnet;

public final class CodexTailnetNative {
    static {
        System.loadLibrary("codexmobile");
        System.loadLibrary("codexmobile_jni");
    }

    private CodexTailnetNative() {}

    public static native String version();

    public static native String status();

    public static native String start(String enrollmentPayloadJson, String stateDir, int tunFd);

    public static native String configureBridgeProxy(String bridgeEndpoint, String authToken);

    public static native String stop();

    public static native void installVpnService(CodexTailnetService service);

    public static native void clearVpnService();

    public static native void setDefaultRouteInterface(String interfaceName);

    public static native void setInterfaceSnapshot(String snapshotJson);
}
