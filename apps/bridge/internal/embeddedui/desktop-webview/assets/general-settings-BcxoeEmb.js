import{o as e}from"./chunk-CFjPhJqf.js";import{t}from"./react-CmOmxWgC.js";import{t as n}from"./preload-helper-Chd9yIcd.js";import{i as r,o as i}from"./message-bus-DcBNa_PD.js";import{Ln as a,bt as o,l as s,yt as c}from"./app-server-manager-hooks-CESABIyE.js";import{Ft as l,Lt as u,Pt as d,Rt as f,Ut as p,Xt as m,Zt as h,_ as g,a as _,bn as v,en as y,gn as b,i as x,in as S,o as C,qt as w,rn as T,t as E,tn as D,ut as O,y as k}from"./app-scope-BIXChNbS.js";import{n as A,t as j}from"./use-is-dark-yFY5r_OJ.js";import{t as M}from"./jsx-runtime-C0i2Pncv.js";import{D as N,g as P,j as F,k as I,t as L,y as R}from"./settings-content-layout-bde-vkZG.js";import{t as z}from"./use-window-type-DfdKjZmY.js";import{t as ee}from"./use-auth-CJLmNq3S.js";import{t as te}from"./use-global-state-DM2zPNaJ.js";import{o as ne}from"./composer-atoms-B5x-TGyA.js";import{a as B,c as re,l as ie}from"./statsig-pbne6FIO.js";import{t as ae}from"./product-event-signal-BC0mz2q_.js";import{a as oe,r as se,t as V}from"./with-window-CbADxG8t.js";import{n as ce}from"./electron-menu-shortcuts-0a_EdMkh.js";import{a as le,i as ue,n as de,r as fe,t as pe}from"./use-is-apps-enabled-zOeYWqoM.js";import{a as me,m as H,r as U,t as he}from"./dropdown-BA4Xat8o.js";import{$ as ge,A as _e,B as ve,C as ye,D as be,E as xe,F as Se,G as Ce,H as we,I as Te,J as Ee,K as De,L as Oe,M as W,N as ke,O as Ae,P as je,Q as Me,R as Ne,S as Pe,U as Fe,V as Ie,W as Le,X as Re,Y as ze,Z as Be,_ as Ve,a as He,b as Ue,f as We,g as Ge,h as Ke,i as qe,j as Je,k as G,l as Ye,n as Xe,o as Ze,q as Qe,r as $e,s as et,t as tt,u as nt,v as rt,w as it,x as at,y as ot,z as st}from"./use-resolved-theme-variant-CLT9YGKw.js";import{n as ct,t as lt}from"./use-os-info-CSaPGU3W.js";import{t as ut}from"./checkbox-ezU3MNxJ.js";import{t as dt}from"./invalidate-queries-and-broadcast-BY5ptdke.js";import{n as ft,t as K}from"./use-configuration-CqgKs048.js";import{a as pt,i as mt,n as ht,r as gt,t as _t}from"./dialog-layout-rR7bsLro.js";import{t as vt}from"./segmented-toggle-Dg8iTnv5.js";import{t as yt}from"./toggle-DDXZ1TeR.js";import{i as bt,n as xt,t as St}from"./popover-BTJ7RlVq.js";import{t as q}from"./settings-row-BFzfJr5B.js";import{n as J,o as Ct,r as wt,t as Tt}from"./settings-surface-Dygq7czW.js";function Et(e){return m(d,e)}var Y=i();function Dt(){let e=(0,Y.c)(3),{authMethod:t}=ee(),[n]=D(`statsig_default_enable_features`),r;return e[0]!==t||e[1]!==n?.fast_mode?(r=n?.fast_mode===!0&&Ot(t),e[0]=t,e[1]=n?.fast_mode,e[2]=r):r=e[2],r}function Ot(e){return e===`chatgpt`}function kt(){return Dt()}function At(e){let t=(0,Y.c)(9),n=c(),i=o(),a=v(E),s;t[0]===n.serviceTier?s=t[1]:(s=k(n.serviceTier),t[0]=n.serviceTier,t[1]=s);let l=s,u;t[2]!==l||t[3]!==a||t[4]!==i?(u=async(e,t)=>{let n=k(e),o=l??`standard`,s=n??`standard`;try{i(n)}catch(e){let t=e;r.error(`Failed to set default service tier`,{safe:{},sensitive:{error:t}});return}a.get(ae).log({eventName:`codex_service_tier_changed`,metadata:{previous_service_tier:o,service_tier:s,source:t}})},t[2]=l,t[3]=a,t[4]=i,t[5]=u):u=t[5];let d=u,f;return t[6]!==n||t[7]!==d?(f={serviceTierSettings:n,setServiceTier:d},t[6]=n,t[7]=d,t[8]=f):f=t[8],f}function jt(e,t){return e?.start===t?.start&&e?.end===t?.end&&e?.side===t?.side&&e?.endSide===t?.endSide}function Mt(){return W({tagName:`button`,properties:{"data-utility-button":``,type:`button`},children:[ke({name:`diffs-icon-plus`,properties:{"data-icon":``}})]})}function Nt(e,t){return e.lineNumber===t.lineNumber&&e.side===t.side}var Pt=class{hoveredLine;pre;gutterUtilityContainer;gutterUtilityButton;gutterUtilitySlot;interactiveLinesAttr=!1;interactiveLineNumbersAttr=!1;hasPointerListeners=!1;hasDocumentPointerListeners=!1;selectedRange=null;renderedSelectionRange;selectionAnchor;queuedSelectionRender;pointerSession={mode:`idle`};constructor(e,t){this.mode=e,this.options=t}setOptions(e){this.options=e}cleanUp(){this.pre?.removeEventListener(`click`,this.handlePointerClick),this.pre?.removeEventListener(`pointerdown`,this.handlePointerDown),this.pre?.removeEventListener(`pointermove`,this.handlePointerMove),this.pre?.removeEventListener(`pointerleave`,this.handlePointerLeave),this.pre?.removeAttribute(`data-interactive-lines`),this.pre?.removeAttribute(`data-interactive-line-numbers`),this.pre=void 0,this.gutterUtilityContainer?.remove(),this.gutterUtilityContainer=void 0,this.gutterUtilityButton=void 0,this.gutterUtilitySlot=void 0,this.clearHoveredLine(),this.detachDocumentPointerListeners(),this.clearPointerSession(),this.queuedSelectionRender!=null&&(cancelAnimationFrame(this.queuedSelectionRender),this.queuedSelectionRender=void 0),this.interactiveLinesAttr=!1,this.interactiveLineNumbersAttr=!1,this.hasPointerListeners=!1}setup(e){this.setSelectionDirty();let{usesCustomGutterUtility:t=!1,enableGutterUtility:n=!1}=this.options;this.pre!==e&&(this.cleanUp(),this.pre=e),n?this.ensureGutterUtilityNode(t):this.gutterUtilityContainer!=null&&(this.gutterUtilityContainer.remove(),this.gutterUtilityContainer=void 0,this.gutterUtilityButton=void 0,this.gutterUtilitySlot=void 0,this.pointerSession.mode===`gutterSelecting`&&(this.clearPointerSession(),this.detachDocumentPointerListeners())),this.syncPointerListeners(e),this.updateInteractiveLineAttributes(),this.renderSelection()}setSelectionDirty(){this.renderedSelectionRange=void 0}isSelectionDirty(){return this.renderedSelectionRange===null}setSelection(e){let t=!(e===this.selectedRange||jt(e??void 0,this.selectedRange??void 0));!this.isSelectionDirty()&&!t||(this.selectedRange=e,this.renderSelection(),t&&this.notifySelectionCommitted())}getSelection(){return this.selectedRange}getHoveredLine=()=>{if(this.hoveredLine!=null){if(this.mode===`diff`&&this.hoveredLine.type===`diff-line`)return{lineNumber:this.hoveredLine.lineNumber,side:this.hoveredLine.annotationSide};if(this.mode===`file`&&this.hoveredLine.type===`line`)return{lineNumber:this.hoveredLine.lineNumber}}};handlePointerClick=e=>{let{onHunkExpand:t,onLineClick:n,onLineNumberClick:r,onMergeConflictActionClick:i}=this.options;t==null&&n==null&&r==null&&i==null||this.options.onGutterUtilityClick!=null&&Ut(e.composedPath())||(X(this.options.__debugPointerEvents,`click`,`FileDiff.DEBUG.handlePointerClick:`,e),this.handlePointerEvent({eventType:`click`,event:e}))};handlePointerMove=e=>{let{lineHoverHighlight:t=`disabled`,onLineEnter:n,onLineLeave:r,enableGutterUtility:i=!1}=this.options;t===`disabled`&&!i&&n==null&&r==null||(X(this.options.__debugPointerEvents,`move`,`FileDiff.DEBUG.handlePointerMove:`,e),this.handlePointerEvent({eventType:`move`,event:e}))};handlePointerLeave=e=>{let{__debugPointerEvents:t}=this.options;if(X(t,`move`,`FileDiff.DEBUG.handlePointerLeave: no event`),this.hoveredLine==null){X(t,`move`,`FileDiff.DEBUG.handlePointerLeave: returned early, no .hoveredLine`);return}this.gutterUtilityContainer?.remove(),this.options.onLineLeave?.({...this.hoveredLine,event:e}),this.clearHoveredLine()};handlePointerEvent({eventType:e,event:t}){let{__debugPointerEvents:n}=this.options,r=t.composedPath();X(n,e,`FileDiff.DEBUG.handlePointerEvent:`,{eventType:e,composedPath:r});let i=this.resolvePointerTarget(r);X(n,e,`FileDiff.DEBUG.handlePointerEvent: resolvePointerTarget result:`,i);let{onLineClick:a,onLineNumberClick:o,onLineEnter:s,onLineLeave:c,onHunkExpand:l,onMergeConflictActionClick:u}=this.options;switch(e){case`move`:if(Lt(i)&&this.hoveredLine?.lineElement===i.lineElement)break;this.hoveredLine!=null&&(this.gutterUtilityContainer?.remove(),c?.({...this.hoveredLine,event:t}),this.clearHoveredLine()),Lt(i)&&(this.setHoveredLine(this.toEventBaseProps(i)),this.gutterUtilityContainer!=null&&i.numberElement.appendChild(this.gutterUtilityContainer),s?.({...this.hoveredLine,event:t}));break;case`click`:{if(i==null)break;if(zt(i)&&u!=null){u(i);break}if(Rt(i)&&l!=null){l(i.hunkIndex,t.shiftKey?`both`:i.direction,t.shiftKey?1/0:void 0);break}if(!Lt(i))break;let e=this.toEventBaseProps(i);o!=null&&i.numberColumn?o({...e,event:t}):a?.({...e,event:t});break}}}syncPointerListeners(e){let{__debugPointerEvents:t,lineHoverHighlight:n=`disabled`,onLineClick:r,onLineNumberClick:i,onLineEnter:a,onLineLeave:o,onHunkExpand:s,onMergeConflictActionClick:c,enableGutterUtility:l=!1,enableLineSelection:u=!1,onGutterUtilityClick:d}=this.options,f=d!=null,p=n!==`disabled`||r!=null||i!=null||s!=null||c!=null||a!=null||o!=null||l||u||f;p&&!this.hasPointerListeners?(e.addEventListener(`click`,this.handlePointerClick),e.addEventListener(`pointerdown`,this.handlePointerDown),e.addEventListener(`pointermove`,this.handlePointerMove),e.addEventListener(`pointerleave`,this.handlePointerLeave),this.hasPointerListeners=!0,X(t,`click`,`FileDiff.DEBUG.attachEventListeners: Attaching click events for:`,(()=>{let e=[];return(t===`both`||t===`click`)&&(r!=null&&e.push(`onLineClick`),i!=null&&e.push(`onLineNumberClick`),s!=null&&e.push(`expandable hunk separators`),c!=null&&e.push(`merge conflict actions`)),e})()),X(t,`move`,`FileDiff.DEBUG.attachEventListeners: Attaching pointer move event`),X(t,`move`,`FileDiff.DEBUG.attachEventListeners: Attaching pointer leave event`)):!p&&this.hasPointerListeners&&(e.removeEventListener(`click`,this.handlePointerClick),e.removeEventListener(`pointerdown`,this.handlePointerDown),e.removeEventListener(`pointermove`,this.handlePointerMove),e.removeEventListener(`pointerleave`,this.handlePointerLeave),this.hasPointerListeners=!1);let m=this.pointerSession.mode===`selecting`||this.pointerSession.mode===`pendingSingleLineUnselect`,h=this.pointerSession.mode===`gutterSelecting`;(!u&&m||!f&&h)&&(this.clearPointerSession(),this.detachDocumentPointerListeners(),this.selectionAnchor=void 0,this.clearPendingSingleLineState())}updateInteractiveLineAttributes(){if(this.pre==null)return;let{onLineClick:e,onLineNumberClick:t,enableLineSelection:n=!1}=this.options,r=e!=null,i=t!=null||n;r&&!this.interactiveLinesAttr?(this.pre.setAttribute(`data-interactive-lines`,``),this.interactiveLinesAttr=!0):!r&&this.interactiveLinesAttr&&(this.pre.removeAttribute(`data-interactive-lines`),this.interactiveLinesAttr=!1),i&&!this.interactiveLineNumbersAttr?(this.pre.setAttribute(`data-interactive-line-numbers`,``),this.interactiveLineNumbersAttr=!0):!i&&this.interactiveLineNumbersAttr&&(this.pre.removeAttribute(`data-interactive-line-numbers`),this.interactiveLineNumbersAttr=!1)}handlePointerDown=e=>{if(e.pointerType===`mouse`&&e.button!==0||this.pre==null||this.pointerSession.mode!==`idle`)return;let t=e.composedPath();Ut(t)&&this.options.onGutterUtilityClick!=null?this.startGutterSelectionFromPointerDown(e,t):this.startLineSelectionFromPointerDown(e,t)};startLineSelectionFromPointerDown(e,t){let{enableLineSelection:n=!1}=this.options;if(!n)return;let r=this.getSelectionPointerInfo(t,!0);if(r==null)return;let{pre:i}=this;if(i==null)return;e.preventDefault();let{lineNumber:a,eventSide:o,lineIndex:s}=r;if(e.shiftKey&&this.selectedRange!=null){let t=this.getIndexesFromSelection(this.selectedRange,i.getAttribute(`data-diff-type`)===`split`);if(t==null)return;let n=t.start<=t.end?s>=t.start:s<=t.end;this.selectionAnchor={lineNumber:n?this.selectedRange.start:this.selectedRange.end,side:n?this.selectedRange.side:this.selectedRange.endSide??this.selectedRange.side},this.updateSelection(a,o,!1),this.notifySelectionStart(this.selectedRange),this.pointerSession={mode:`selecting`,pointerId:e.pointerId},this.attachDocumentPointerListeners();return}if(this.selectedRange?.start===a&&this.selectedRange?.end===a){let t={lineNumber:a,side:o};this.selectionAnchor=t,this.pointerSession={mode:`pendingSingleLineUnselect`,pointerId:e.pointerId,anchor:t,pending:t},this.attachDocumentPointerListeners();return}this.selectedRange=null,this.selectionAnchor={lineNumber:a,side:o},this.updateSelection(a,o,!1),this.notifySelectionStart(this.selectedRange),this.pointerSession={mode:`selecting`,pointerId:e.pointerId},this.attachDocumentPointerListeners()}startGutterSelectionFromPointerDown(e,t){let{enableLineSelection:n=!1,onGutterUtilityClick:r}=this.options;if(r==null)return;let i=this.getSelectionPointFromPath(t);i!=null&&(e.preventDefault(),e.stopPropagation(),this.pointerSession={mode:`gutterSelecting`,pointerId:e.pointerId,anchor:i,current:i},n&&(this.selectionAnchor={lineNumber:i.lineNumber,side:i.side},this.updateSelection(i.lineNumber,i.side,!1),this.notifySelectionStart(this.selectedRange)),this.attachDocumentPointerListeners())}handleDocumentPointerMove=e=>{let{enableLineSelection:t=!1}=this.options;switch(this.pointerSession.mode){case`idle`:return;case`gutterSelecting`:{if(e.pointerId!==this.pointerSession.pointerId)return;let n=this.getSelectionPointFromPath(e.composedPath());if(n==null)return;this.pointerSession.current=n,t===!0&&this.updateSelection(n.lineNumber,n.side);return}case`selecting`:{if(e.pointerId!==this.pointerSession.pointerId)return;let t=this.getSelectionPointerInfo(e.composedPath(),!1);if(t==null||this.selectionAnchor==null)return;this.updateSelection(t.lineNumber,t.eventSide);return}case`pendingSingleLineUnselect`:{if(e.pointerId!==this.pointerSession.pointerId)return;let t=this.getSelectionPointerInfo(e.composedPath(),!1);if(t==null||this.selectionAnchor==null)return;let n={lineNumber:t.lineNumber,side:t.eventSide};if(Nt(this.pointerSession.pending,n))return;this.updateSelection(t.lineNumber,t.eventSide,!1),this.notifySelectionStart(this.selectedRange),this.notifySelectionChangeDelta(),this.pointerSession={mode:`selecting`,pointerId:e.pointerId};return}}};handleDocumentPointerUp=e=>{let{enableLineSelection:t=!1,onGutterUtilityClick:n}=this.options;switch(this.pointerSession.mode){case`idle`:return;case`gutterSelecting`:{if(e.pointerId!==this.pointerSession.pointerId)return;let r=this.getSelectionPointFromPath(e.composedPath());r!=null&&(this.pointerSession.current=r,t&&this.updateSelection(r.lineNumber,r.side)),n?.(this.buildSelectedLineRange(this.pointerSession.anchor,this.pointerSession.current)),this.selectionAnchor=void 0,t&&(this.notifySelectionEnd(this.selectedRange),this.notifySelectionCommitted()),this.clearPointerSession(),this.detachDocumentPointerListeners();return}case`pendingSingleLineUnselect`:if(e.pointerId!==this.pointerSession.pointerId)return;this.updateSelection(null,void 0,!1),this.selectionAnchor=void 0,this.clearPendingSingleLineState(),this.detachDocumentPointerListeners(),this.notifySelectionEnd(this.selectedRange),this.notifySelectionCommitted();return;case`selecting`:if(e.pointerId!==this.pointerSession.pointerId)return;this.selectionAnchor=void 0,this.detachDocumentPointerListeners(),this.clearPointerSession(),this.notifySelectionEnd(this.selectedRange),this.notifySelectionCommitted()}};handleDocumentPointerCancel=e=>{switch(this.pointerSession.mode){case`idle`:return;case`gutterSelecting`:case`selecting`:case`pendingSingleLineUnselect`:if(`pointerId`in this.pointerSession&&e.pointerId!==this.pointerSession.pointerId)return;this.selectionAnchor=void 0,this.clearPendingSingleLineState(),this.clearPointerSession(),this.detachDocumentPointerListeners()}};clearHoveredLine(){this.hoveredLine!=null&&(this.hoveredLine.lineElement.removeAttribute(`data-hovered`),this.hoveredLine.numberElement.removeAttribute(`data-hovered`),this.hoveredLine=void 0)}setHoveredLine(e){let{lineHoverHighlight:t=`disabled`}=this.options;this.hoveredLine!=null&&this.clearHoveredLine(),this.hoveredLine=e,t!==`disabled`&&((t===`both`||t===`line`)&&this.hoveredLine.lineElement.setAttribute(`data-hovered`,``),(t===`both`||t===`number`)&&this.hoveredLine.numberElement.setAttribute(`data-hovered`,``))}ensureGutterUtilityNode(e){if(this.gutterUtilityContainer??(this.gutterUtilityContainer=document.createElement(`div`),this.gutterUtilityContainer.setAttribute(`data-gutter-utility-slot`,``)),e)this.gutterUtilityButton!=null&&(this.gutterUtilityButton.remove(),this.gutterUtilityButton=void 0),this.gutterUtilitySlot??(this.gutterUtilitySlot=document.createElement(`slot`),this.gutterUtilitySlot.name=`gutter-utility-slot`),this.gutterUtilitySlot.parentNode!==this.gutterUtilityContainer&&this.gutterUtilityContainer.replaceChildren(this.gutterUtilitySlot);else{if(this.gutterUtilitySlot?.remove(),this.gutterUtilitySlot=void 0,this.gutterUtilityButton==null){let e=document.createElement(`div`);e.innerHTML=Ae(Mt());let t=e.firstElementChild;if(!(t instanceof HTMLButtonElement))throw Error(`InteractionManager.ensureGutterUtilityNode: Node element should be a button`);t.remove(),this.gutterUtilityButton=t}this.gutterUtilityButton.parentNode!==this.gutterUtilityContainer&&this.gutterUtilityContainer.replaceChildren(this.gutterUtilityButton)}}attachDocumentPointerListeners(){this.hasDocumentPointerListeners||=(document.addEventListener(`pointermove`,this.handleDocumentPointerMove),document.addEventListener(`pointerup`,this.handleDocumentPointerUp),document.addEventListener(`pointercancel`,this.handleDocumentPointerCancel),!0)}detachDocumentPointerListeners(){this.hasDocumentPointerListeners&&=(document.removeEventListener(`pointermove`,this.handleDocumentPointerMove),document.removeEventListener(`pointerup`,this.handleDocumentPointerUp),document.removeEventListener(`pointercancel`,this.handleDocumentPointerCancel),!1)}clearPointerSession(){this.pointerSession={mode:`idle`}}clearPendingSingleLineState(){this.pointerSession.mode===`pendingSingleLineUnselect`&&(this.pointerSession={mode:`idle`})}getSelectionPointerInfo(e,t){let n=this.resolvePointerTarget(e);if(Lt(n)&&!(t&&!n.numberColumn)&&n.splitLineIndex!=null)return{lineIndex:n.splitLineIndex,lineNumber:n.lineNumber,eventSide:this.mode===`diff`?n.side:void 0}}getSelectionPointFromPath(e){let t=this.resolvePointerTarget(e);if(Lt(t))return{lineNumber:t.lineNumber,side:this.mode===`diff`?t.side:void 0}}getLineIndex(e,t){let{getLineIndex:n}=this.options;return n==null?[e-1,e-1]:n(e,t)}updateSelection(e,t,n=!0){let{selectedRange:r}=this,i;if(e==null)i=null;else{let n=this.selectionAnchor?.side??t,r=this.selectionAnchor?.lineNumber??e;i=this.buildSelectionRange(r,e,n,t)}jt(r??void 0,i??void 0)||(this.selectedRange=i,n&&this.notifySelectionChangeDelta(),this.queuedSelectionRender??=requestAnimationFrame(this.renderSelection))}getIndexesFromSelection(e,t){if(this.pre==null)return;let n=this.getLineIndex(e.start,e.side),r=this.getLineIndex(e.end,e.endSide??e.side);return n!=null&&r!=null?{start:t?n[1]:n[0],end:t?r[1]:r[0]}:void 0}renderSelection=()=>{if(this.queuedSelectionRender!=null&&(cancelAnimationFrame(this.queuedSelectionRender),this.queuedSelectionRender=void 0),this.pre==null||this.renderedSelectionRange===this.selectedRange)return;let e=this.pre.querySelectorAll(`[data-selected-line]`);for(let t of e)t.removeAttribute(`data-selected-line`);if(this.renderedSelectionRange=this.selectedRange,this.selectedRange==null)return;let{children:t}=this.pre;if(t.length===0)return;if(t.length>2)throw console.error(t),Error(`InteractionManager.renderSelection: Somehow there are more than 2 code elements...`);let n=this.pre.getAttribute(`data-diff-type`)===`split`,r=this.getIndexesFromSelection(this.selectedRange,n);if(r==null)throw console.error({rowRange:r,selectedRange:this.selectedRange}),Error(`InteractionManager.renderSelection: No valid rowRange`);let i=r.start===r.end,a=Math.min(r.start,r.end),o=Math.max(r.start,r.end);for(let e of t){let[t,r]=e.children,s=r.children.length;if(s!==t.children.length)throw Error(`InteractionManager.renderSelection: gutter and content children dont match, something is wrong`);for(let e=0;e<s;e++){let s=r.children[e],c=t.children[e];if(!(s instanceof HTMLElement)||!(c instanceof HTMLElement))continue;let l=this.parseLineIndex(s,n);if((l??0)>o)break;if(l==null||l<a)continue;let u=i?`single`:l===a?`first`:l===o?`last`:``;s.setAttribute(`data-selected-line`,u),c.setAttribute(`data-selected-line`,u),c.nextSibling instanceof HTMLElement&&s.nextSibling instanceof HTMLElement&&(s.nextSibling.hasAttribute(`data-line-annotation`)||s.nextSibling.hasAttribute(`data-merge-conflict-actions`))&&(i?(u=`last`,s.setAttribute(`data-selected-line`,`first`)):l===a?u=``:l===o&&s.setAttribute(`data-selected-line`,``),s.nextSibling.setAttribute(`data-selected-line`,u),c.nextSibling.setAttribute(`data-selected-line`,u))}}};notifySelectionCommitted(){this.options.onLineSelected?.(this.selectedRange??null)}notifySelectionChangeDelta(){this.options.onLineSelectionChange?.(this.selectedRange??null)}notifySelectionStart(e){this.options.onLineSelectionStart?.(e)}notifySelectionEnd(e){this.options.onLineSelectionEnd?.(e)}toEventBaseProps(e){return this.mode===`file`?{type:`line`,lineElement:e.lineElement,lineNumber:e.lineNumber,numberColumn:e.numberColumn,numberElement:e.numberElement}:{type:`diff-line`,annotationSide:e.side,lineType:e.lineType,lineElement:e.lineElement,numberElement:e.numberElement,lineNumber:e.lineNumber,numberColumn:e.numberColumn}}buildSelectedLineRange(e,t){return this.buildSelectionRange(e.lineNumber,t.lineNumber,e.side,t.side)}buildSelectionRange(e,t,n,r){return{start:e,end:t,...n==null?{}:{side:n},...n!==r&&r!=null?{endSide:r}:{}}}resolvePointerTarget(e){let t=!1,n,r,i,a,o,s,c,l;for(let u of e){if(!(u instanceof HTMLElement))continue;if(l==null&&u.hasAttribute(`data-merge-conflict-action`)){let e=u.getAttribute(`data-merge-conflict-action`)??void 0,t=u.getAttribute(`data-merge-conflict-conflict-index`)??void 0,n=t==null?NaN:Number.parseInt(t,10);Bt(e)&&Number.isFinite(n)&&(l={kind:`merge-conflict-action`,resolution:e,conflictIndex:n})}let e=o==null?u.getAttribute(`data-column-number`)??void 0:void 0;if(e!=null){o=u,c=Number.parseInt(e,10),t=!0,n=Ht(u),a=u.getAttribute(`data-line-index`)??void 0;continue}let d=i==null?u.getAttribute(`data-line`)??void 0:void 0;if(d!=null){i=u,c=Number.parseInt(d,10),n=Ht(u),a=u.getAttribute(`data-line-index`)??void 0;continue}if(s==null&&u.hasAttribute(`data-expand-button`)){s={hunkIndex:void 0,direction:u.hasAttribute(`data-expand-up`)?`up`:u.hasAttribute(`data-expand-down`)?`down`:`both`};continue}let f=s==null?void 0:u.getAttribute(`data-expand-index`)??void 0;if(s!=null&&f!=null){let e=Number.parseInt(f,10);Number.isNaN(e)||(s.hunkIndex=e);continue}if(r==null&&u.hasAttribute(`data-code`)){r=u;break}}if(l!=null)return l;if(s?.hunkIndex!=null)return{type:`line-info`,hunkIndex:s.hunkIndex,direction:s.direction};if(i??=a==null?void 0:Vt(r,`[data-line][data-line-index="${a}"]`),o??=a==null?void 0:Vt(r,`[data-column-number][data-line-index="${a}"]`),r==null||i==null||o==null||n==null||c==null||Number.isNaN(c))return;let u=this.parseLineIndex(i,this.isSplitDiff());if(this.mode===`file`)return{kind:`line`,lineType:n,lineElement:i,lineNumber:c,numberColumn:t,numberElement:o,side:void 0,splitLineIndex:u};let d=(()=>{switch(n){case`change-deletion`:return`deletions`;case`change-addition`:return`additions`;default:return r.hasAttribute(`data-deletions`)?`deletions`:`additions`}})();return{kind:`line`,lineType:n,lineElement:i,lineNumber:c,numberColumn:t,numberElement:o,side:d,splitLineIndex:u}}isSplitDiff(){return this.pre?.getAttribute(`data-diff-type`)===`split`}parseLineIndex(e,t){let n=(e.getAttribute(`data-line-index`)??``).split(`,`).map(e=>Number.parseInt(e,10)).filter(e=>!Number.isNaN(e));if(t&&n.length===2)return n[1];if(!t)return n[0]}};function Ft({enableGutterUtility:e,enableHoverUtility:t,lineHoverHighlight:n,onGutterUtilityClick:r,onLineClick:i,onLineEnter:a,onLineLeave:o,onLineNumberClick:s,renderGutterUtility:c,renderHoverUtility:l,__debugPointerEvents:u,enableLineSelection:d,onLineSelected:f,onLineSelectionStart:p,onLineSelectionChange:m,onLineSelectionEnd:h},g,_,v){return{enableGutterUtility:It({enableGutterUtility:e,enableHoverUtility:t,renderGutterUtility:c,renderHoverUtility:l,onGutterUtilityClick:r}),usesCustomGutterUtility:c!=null||l!=null,lineHoverHighlight:n,onGutterUtilityClick:r,onHunkExpand:g,onMergeConflictActionClick:v,onLineClick:i,onLineEnter:a,onLineLeave:o,onLineNumberClick:s,__debugPointerEvents:u,enableLineSelection:d,onLineSelected:f,onLineSelectionStart:p,onLineSelectionChange:m,onLineSelectionEnd:h,getLineIndex:_}}function It({enableGutterUtility:e,enableHoverUtility:t,renderGutterUtility:n,renderHoverUtility:r,onGutterUtilityClick:i}){if(e!==void 0&&t!==void 0)throw Error(`Cannot use both 'enableGutterUtility' and deprecated 'enableHoverUtility'. Use only 'enableGutterUtility'.`);if(n!=null&&r!=null)throw Error(`Cannot use both 'renderGutterUtility' and deprecated 'renderHoverUtility'. Use only 'renderGutterUtility'.`);if(i!=null&&(n!=null||r!=null))throw Error(`Cannot use both 'onGutterUtilityClick' and render utility callbacks ('renderGutterUtility'/'renderHoverUtility'). Use only one gutter utility API.`);return e??t??!1}function Lt(e){return e!=null&&`kind`in e&&e.kind===`line`}function Rt(e){return`type`in e&&e.type===`line-info`}function zt(e){return`kind`in e&&e.kind===`merge-conflict-action`}function Bt(e){return e===`current`||e===`incoming`||e===`both`}function Vt(e,t){let n=e?.querySelector(t);return n instanceof HTMLElement?n:void 0}function Ht(e){let t=e.getAttribute(`data-line-type`);if(t!=null)switch(t){case`change-deletion`:case`change-addition`:case`context`:case`context-expanded`:return t;default:return}}function Ut(e){for(let t of e)if(t instanceof HTMLElement&&t.hasAttribute(`data-utility-button`))return!0;return!1}function X(e=`none`,t,...n){switch(e){case`none`:return;case`both`:break;case`click`:if(t!==`click`)return;break;case`move`:if(t!==`move`)return;break}console.log(...n)}var Wt=class{observedNodes=new Map;queuedUpdates=new Map;cleanUp(){this.resizeObserver?.disconnect(),this.observedNodes.clear(),this.queuedUpdates.clear()}resizeObserver;setup(e,t){this.resizeObserver??=new ResizeObserver(this.handleResizeObserver);let n=e.querySelectorAll(`code`),r=new Map(this.observedNodes);this.observedNodes.clear();for(let e of n){let t=r.get(e);if(t!=null&&t.type!==`code`)throw Error(`ResizeManager.setup: somehow a code node is being used for an annotation, should be impossible`);let n=e.firstElementChild;n instanceof HTMLElement||(n=null),t==null?(t={type:`code`,codeElement:e,numberElement:n,codeWidth:`auto`,numberWidth:0},this.observedNodes.set(e,t),this.resizeObserver.observe(e),n!=null&&(this.observedNodes.set(n,t),this.resizeObserver.observe(n))):(this.observedNodes.set(e,t),r.delete(e),t.numberElement===n?t.numberElement!=null&&(r.delete(t.numberElement),this.observedNodes.set(t.numberElement,t)):(t.numberElement!=null&&this.resizeObserver.unobserve(t.numberElement),n!=null&&(this.resizeObserver.observe(n),r.delete(n),this.observedNodes.set(n,t)),t.numberElement=n))}if(n.length>1&&!t){let t=e.querySelectorAll(`[data-line-annotation*=","]`),n=new Map;for(let e of t){if(!(e instanceof HTMLElement))continue;let{lineAnnotation:t=``}=e.dataset;if(!/^\d+,\d+$/.test(t)){console.error(`DiffFileRenderer.setupResizeObserver: Invalid element or annotation`,{lineAnnotation:t,element:e});continue}let r=n.get(t);r??(r=[],n.set(t,r)),r.push(e)}for(let[e,t]of n){if(t.length!==2){console.error(`DiffFileRenderer.setupResizeObserver: Bad Pair`,e,t);continue}let[n,i]=t,a=n.firstElementChild,o=i.firstElementChild;if(!(n instanceof HTMLElement)||!(i instanceof HTMLElement)||!(a instanceof HTMLElement)||!(o instanceof HTMLElement))continue;let s=r.get(a);if(s!=null){this.observedNodes.set(a,s),this.observedNodes.set(o,s),r.delete(a),r.delete(o);continue}s={type:`annotations`,column1:{container:n,child:a,childHeight:a.getBoundingClientRect().height},column2:{container:i,child:o,childHeight:o.getBoundingClientRect().height},currentHeight:`auto`};let c=Math.max(s.column1.childHeight,s.column2.childHeight);this.applyNewHeight(s,c),this.observedNodes.set(a,s),this.observedNodes.set(o,s),this.resizeObserver.observe(a),this.resizeObserver.observe(o)}}for(let e of r.keys())e.isConnected&&(e.style.removeProperty(`--diffs-column-content-width`),e.style.removeProperty(`--diffs-column-number-width`),e.style.removeProperty(`--diffs-column-width`),e.parentElement instanceof HTMLElement&&e.parentElement.style.removeProperty(`--diffs-annotation-min-height`)),this.resizeObserver.unobserve(e);r.clear()}handleResizeObserver=e=>{for(let t of e){let{target:e,borderBoxSize:n}=t;if(!(e instanceof HTMLElement)){console.error(`FileDiff.handleResizeObserver: Invalid element for ResizeObserver`,t);continue}let r=this.observedNodes.get(e);if(r==null){console.error(`FileDiff.handleResizeObserver: Not a valid observed node`,t);continue}let i=n[0];if(r.type===`annotations`){let t=(()=>{if(e===r.column1.child)return r.column1;if(e===r.column2.child)return r.column2})();if(t==null){console.error(`FileDiff.handleResizeObserver: Couldn't find a column for`,{item:r,target:e});continue}t.childHeight=i.blockSize;let n=Math.max(r.column1.childHeight,r.column2.childHeight);this.applyNewHeight(r,n)}else if(r.type===`code`){let t=[e,i.inlineSize],n=this.queuedUpdates.get(r)??[];n.push(t),this.queuedUpdates.set(r,n)}}this.handleColumnChange()};handleColumnChange=()=>{for(let[e,t]of this.queuedUpdates)for(let[n,r]of t)if(n===e.codeElement){let n=Math.max(Math.floor(r),0);if(n!==e.codeWidth){let t=Math.max(n-e.numberWidth,0);e.codeWidth=n===0?`auto`:n,e.codeElement.style.setProperty(`--diffs-column-content-width`,`${t>0?`${t}px`:`auto`}`),e.codeElement.style.setProperty(`--diffs-column-width`,`${typeof e.codeWidth==`number`?`${e.codeWidth}px`:`auto`}`)}e.numberElement!=null&&typeof e.codeWidth==`number`&&e.numberWidth===0&&t.push([e.numberElement,e.numberElement.getBoundingClientRect().width])}else if(n===e.numberElement){let t=Math.max(Math.ceil(r),0);if(t!==e.numberWidth&&(e.numberWidth=t,e.codeElement.style.setProperty(`--diffs-column-number-width`,`${e.numberWidth===0?`auto`:`${e.numberWidth}px`}`),e.codeWidth!==`auto`)){let t=Math.max(e.codeWidth-e.numberWidth,0);e.codeElement.style.setProperty(`--diffs-column-content-width`,`${t===0?`auto`:`${t}px`}`)}}this.queuedUpdates.clear()};applyNewHeight(e,t){t!==e.currentHeight&&(e.currentHeight=Math.max(t,0),e.column1.container.style.setProperty(`--diffs-annotation-min-height`,`${e.currentHeight}px`),e.column2.container.style.setProperty(`--diffs-annotation-min-height`,`${e.currentHeight}px`))}};function Gt(e){for(let t of Array.isArray(e)?e:[e])if(!be.has(t))return!1;return!0}function Kt(e){for(let t of Pe(e))if(!xe.has(t))return!1;return!0}function qt(e,t){return e==null||t==null?e===t:e.startingLine===t.startingLine&&e.totalLines===t.totalLines&&e.bufferBefore===t.bufferBefore&&e.bufferAfter===t.bufferAfter}function Jt(e){return W({tagName:`div`,children:[W({tagName:`div`,children:e.annotations?.map(e=>W({tagName:`slot`,properties:{name:e}})),properties:{"data-annotation-content":``}})],properties:{"data-line-annotation":`${e.hunkIndex},${e.lineIndex}`}})}function Yt(e){switch(e){case`file`:return`diffs-icon-file-code`;case`change`:return`diffs-icon-symbol-modified`;case`new`:return`diffs-icon-symbol-added`;case`deleted`:return`diffs-icon-symbol-deleted`;case`rename-pure`:case`rename-changed`:return`diffs-icon-symbol-moved`}}function Xt({fileOrDiff:e,themeStyles:t,themeType:n}){let r=`type`in e?e:void 0,i={"data-diffs-header":``,"data-change-type":r?.type,"data-theme-type":n===`system`?void 0:n,style:t};return W({tagName:`div`,children:[Zt({name:e.name,prevName:`prevName`in e?e.prevName:void 0,iconType:r?.type??`file`}),Qt(r)],properties:i})}function Zt({name:e,prevName:t,iconType:n}){let r=[W({tagName:`slot`,properties:{name:Ee}}),ke({name:Yt(n),properties:{"data-change-icon":n}})];return t!=null&&(r.push(W({tagName:`div`,children:[je(t)],properties:{"data-prev-name":``}})),r.push(ke({name:`diffs-icon-arrow-right-short`,properties:{"data-rename-icon":``}}))),r.push(W({tagName:`div`,children:[je(e)],properties:{"data-title":``}})),W({tagName:`div`,children:r,properties:{"data-header-content":``}})}function Qt(e){let t=[];if(e!=null){let n=0,r=0;for(let t of e.hunks)n+=t.additionLines,r+=t.deletionLines;(r>0||n===0)&&t.push(W({tagName:`span`,children:[je(`-${r}`)],properties:{"data-deletions-count":``}})),(n>0||r===0)&&t.push(W({tagName:`span`,children:[je(`+${n}`)],properties:{"data-additions-count":``}}))}return t.push(W({tagName:`slot`,properties:{name:Qe}})),W({tagName:`div`,children:t,properties:{"data-metadata":``}})}function $t(e){return W({tagName:`pre`,properties:en(e)})}function en({diffIndicators:e,disableBackground:t,disableLineNumbers:n,overflow:r,split:i,themeType:a,themeStyles:o,totalLines:s,type:c,customProperties:l}){let u={...l,"data-diff":c===`diff`?``:void 0,"data-file":c===`file`?``:void 0,"data-diff-type":c===`diff`?i?`split`:`single`:void 0,"data-overflow":r,"data-disable-line-numbers":n?``:void 0,"data-background":t?void 0:``,"data-indicators":e===`bars`||e===`classic`?e:void 0,"data-theme-type":a===`system`?void 0:a,style:o,tabIndex:0};return u.style+=`--diffs-min-number-column-width-default:${`${s}`.length}ch;`,u}function tn(e,{theme:t,preferredHighlighter:n=`shiki-js`}){return{langs:[e??`text`],themes:Pe(t),preferredHighlighter:n}}function nn(e){return`annotation-${`side`in e?`${e.side}-`:``}${e.lineNumber}`}function rn(e,t){return W({tagName:`div`,children:e,properties:{"data-content":``,style:`grid-row: span ${t}`}})}var an=`<svg data-icon-sprite aria-hidden="true" width="0" height="0">
  <symbol id="diffs-icon-arrow-right-short" viewBox="0 0 16 16">
    <path d="M8.47 4.22a.75.75 0 0 0 0 1.06l1.97 1.97H3.75a.75.75 0 0 0 0 1.5h6.69l-1.97 1.97a.75.75 0 1 0 1.06 1.06l3.25-3.25a.75.75 0 0 0 0-1.06L9.53 4.22a.75.75 0 0 0-1.06 0"/>
  </symbol>
  <symbol id="diffs-icon-brand-github" viewBox="0 0 16 16">
    <path d="M8 0c4.42 0 8 3.58 8 8a8.01 8.01 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27s-1.36.09-2 .27c-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8"/>
  </symbol>
  <symbol id="diffs-icon-chevron" viewBox="0 0 16 16">
    <path d="M1.47 4.47a.75.75 0 0 1 1.06 0L8 9.94l5.47-5.47a.75.75 0 1 1 1.06 1.06l-6 6a.75.75 0 0 1-1.06 0l-6-6a.75.75 0 0 1 0-1.06"/>
  </symbol>
  <symbol id="diffs-icon-chevrons-narrow" viewBox="0 0 10 16">
    <path d="M4.47 2.22a.75.75 0 0 1 1.06 0l3.25 3.25a.75.75 0 0 1-1.06 1.06L5 3.81 2.28 6.53a.75.75 0 0 1-1.06-1.06zM1.22 9.47a.75.75 0 0 1 1.06 0L5 12.19l2.72-2.72a.75.75 0 0 1 1.06 1.06l-3.25 3.25a.75.75 0 0 1-1.06 0l-3.25-3.25a.75.75 0 0 1 0-1.06"/>
  </symbol>
  <symbol id="diffs-icon-diff-split" viewBox="0 0 16 16">
    <path d="M14 0H8.5v16H14a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2m-1.5 6.5v1h1a.5.5 0 0 1 0 1h-1v1a.5.5 0 0 1-1 0v-1h-1a.5.5 0 0 1 0-1h1v-1a.5.5 0 0 1 1 0"/><path d="M2 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h5.5V0zm.5 7.5h3a.5.5 0 0 1 0 1h-3a.5.5 0 0 1 0-1" opacity=".3"/>
  </symbol>
  <symbol id="diffs-icon-diff-unified" viewBox="0 0 16 16">
    <path fill-rule="evenodd" d="M16 14a2 2 0 0 1-2 2H2a2 2 0 0 1-2-2V8.5h16zm-8-4a.5.5 0 0 0-.5.5v1h-1a.5.5 0 0 0 0 1h1v1a.5.5 0 0 0 1 0v-1h1a.5.5 0 0 0 0-1h-1v-1A.5.5 0 0 0 8 10" clip-rule="evenodd"/><path fill-rule="evenodd" d="M14 0a2 2 0 0 1 2 2v5.5H0V2a2 2 0 0 1 2-2zM6.5 3.5a.5.5 0 0 0 0 1h3a.5.5 0 0 0 0-1z" clip-rule="evenodd" opacity=".4"/>
  </symbol>
  <symbol id="diffs-icon-expand" viewBox="0 0 16 16">
    <path d="M3.47 5.47a.75.75 0 0 1 1.06 0L8 8.94l3.47-3.47a.75.75 0 1 1 1.06 1.06l-4 4a.75.75 0 0 1-1.06 0l-4-4a.75.75 0 0 1 0-1.06"/>
  </symbol>
  <symbol id="diffs-icon-expand-all" viewBox="0 0 16 16">
    <path d="M11.47 9.47a.75.75 0 1 1 1.06 1.06l-4 4a.75.75 0 0 1-1.06 0l-4-4a.75.75 0 1 1 1.06-1.06L8 12.94zM7.526 1.418a.75.75 0 0 1 1.004.052l4 4a.75.75 0 1 1-1.06 1.06L8 3.06 4.53 6.53a.75.75 0 1 1-1.06-1.06l4-4z"/>
  </symbol>
  <symbol id="diffs-icon-file-code" viewBox="0 0 16 16">
    <path d="M10.75 0c.199 0 .39.08.53.22l3.5 3.5c.14.14.22.331.22.53v9A2.75 2.75 0 0 1 12.25 16h-8.5A2.75 2.75 0 0 1 1 13.25V2.75A2.75 2.75 0 0 1 3.75 0zm-7 1.5c-.69 0-1.25.56-1.25 1.25v10.5c0 .69.56 1.25 1.25 1.25h8.5c.69 0 1.25-.56 1.25-1.25V5h-1.25A2.25 2.25 0 0 1 10 2.75V1.5z"/><path d="M7.248 6.19a.75.75 0 0 1 .063 1.058L5.753 9l1.558 1.752a.75.75 0 0 1-1.122.996l-2-2.25a.75.75 0 0 1 0-.996l2-2.25a.75.75 0 0 1 1.06-.063M8.69 7.248a.75.75 0 1 1 1.12-.996l2 2.25a.75.75 0 0 1 0 .996l-2 2.25a.75.75 0 1 1-1.12-.996L10.245 9z"/>
  </symbol>
  <symbol id="diffs-icon-plus" viewBox="0 0 16 16">
    <path d="M8 3a.75.75 0 0 1 .75.75v3.5h3.5a.75.75 0 0 1 0 1.5h-3.5v3.5a.75.75 0 0 1-1.5 0v-3.5h-3.5a.75.75 0 0 1 0-1.5h3.5v-3.5A.75.75 0 0 1 8 3"/>
  </symbol>
  <symbol id="diffs-icon-symbol-added" viewBox="0 0 16 16">
    <path d="M8 4a.75.75 0 0 1 .75.75v2.5h2.5a.75.75 0 0 1 0 1.5h-2.5v2.5a.75.75 0 0 1-1.5 0v-2.5h-2.5a.75.75 0 0 1 0-1.5h2.5v-2.5A.75.75 0 0 1 8 4"/><path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/>
  </symbol>
  <symbol id="diffs-icon-symbol-deleted" viewBox="0 0 16 16">
    <path d="M4 8a.75.75 0 0 1 .75-.75h6.5a.75.75 0 0 1 0 1.5h-6.5A.75.75 0 0 1 4 8"/><path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/>
  </symbol>
  <symbol id="diffs-icon-symbol-diffstat" viewBox="0 0 16 16">
    <path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/><path d="M8.75 4.296a.75.75 0 0 0-1.5 0V6.25h-2a.75.75 0 0 0 0 1.5h2v1.5h1.5v-1.5h2a.75.75 0 0 0 0-1.5h-2zM5.25 10a.75.75 0 0 0 0 1.5h5.5a.75.75 0 0 0 0-1.5z"/>
  </symbol>
  <symbol id="diffs-icon-symbol-ignored" viewBox="0 0 16 16">
    <path d="M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706s.826.607 1.706.802c.898.2 2.091.288 3.704.288s2.806-.088 3.704-.288c.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5s-2.806.088-3.704.288c-.88.196-1.381.478-1.706.802s-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8M0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m11.53-2.47a.75.75 0 0 0-1.06-1.06l-6 6a.75.75 0 1 0 1.06 1.06z"/>
  </symbol>
  <symbol id="diffs-icon-symbol-modified" viewBox="0 0 16 16">
    <path d="M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706s.826.607 1.706.802c.898.2 2.091.288 3.704.288s2.806-.088 3.704-.288c.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5s-2.806.088-3.704.288c-.88.196-1.381.478-1.706.802s-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8M0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m8 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6"/>
  </symbol>
  <symbol id="diffs-icon-symbol-moved" viewBox="0 0 16 16">
    <path d="M1.788 4.296c.196-.88.478-1.381.802-1.706s.826-.606 1.706-.802C5.194 1.588 6.387 1.5 8 1.5s2.806.088 3.704.288c.88.196 1.381.478 1.706.802s.607.826.802 1.706c.2.898.288 2.091.288 3.704s-.088 2.806-.288 3.704c-.195.88-.478 1.381-.802 1.706s-.826.607-1.706.802c-.898.2-2.091.288-3.704.288s-2.806-.088-3.704-.288c-.88-.195-1.381-.478-1.706-.802s-.606-.826-.802-1.706C1.588 10.806 1.5 9.613 1.5 8s.088-2.806.288-3.704M8 0C1.412 0 0 1.412 0 8s1.412 8 8 8 8-1.412 8-8-1.412-8-8-8"/><path d="M8.495 4.695a.75.75 0 0 0-.05 1.06L10.486 8l-2.041 2.246a.75.75 0 0 0 1.11 1.008l2.5-2.75a.75.75 0 0 0 0-1.008l-2.5-2.75a.75.75 0 0 0-1.06-.051m-4 0a.75.75 0 0 0-.05 1.06l2.044 2.248-1.796 1.995a.75.75 0 0 0 1.114 1.004l2.25-2.5a.75.75 0 0 0-.002-1.007l-2.5-2.75a.75.75 0 0 0-1.06-.05"/>
  </symbol>
  <symbol id="diffs-icon-symbol-ref" viewBox="0 0 16 16">
    <path d="M1.5 8c0 1.613.088 2.806.288 3.704.196.88.478 1.381.802 1.706.286.286.71.54 1.41.73V1.86c-.7.19-1.124.444-1.41.73-.324.325-.606.826-.802 1.706C1.588 5.194 1.5 6.387 1.5 8m4 6.397c.697.07 1.522.103 2.5.103 1.613 0 2.806-.088 3.704-.288.88-.195 1.381-.478 1.706-.802s.607-.826.802-1.706c.2-.898.288-2.091.288-3.704s-.088-2.806-.288-3.704c-.195-.88-.478-1.381-.802-1.706s-.826-.606-1.706-.802C10.806 1.588 9.613 1.5 8 1.5c-.978 0-1.803.033-2.5.103zM0 8c0-6.588 1.412-8 8-8s8 1.412 8 8-1.412 8-8 8-8-1.412-8-8m7-2a1 1 0 0 1 1-1h3a1 1 0 0 1 1 1v1a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1z"/>
  </symbol>
</svg>`;function on(e,t){return e==null||t==null?e===t:cn(e.customProperties,t.customProperties)&&e.type===t.type&&e.diffIndicators===t.diffIndicators&&e.disableBackground===t.disableBackground&&e.disableLineNumbers===t.disableLineNumbers&&e.overflow===t.overflow&&e.split===t.split&&e.themeStyles===t.themeStyles&&e.themeType===t.themeType&&e.totalLines===t.totalLines}var sn={};function cn(e=sn,t=sn){if(e===t)return!0;let n=Object.keys(e),r=Object.keys(t);if(n.length!==r.length)return!1;for(let r of n)if(e[r]!==t[r])return!1;return!0}function ln(e){let t=document.createElement(`div`);return t.dataset.annotationSlot=``,t.slot=e,t.style.whiteSpace=`normal`,t}function un(){let e=document.createElement(`div`);return e.slot=`gutter-utility-slot`,e.style.position=`absolute`,e.style.top=`0`,e.style.bottom=`0`,e.style.textAlign=`center`,e.style.whiteSpace=`normal`,e}function dn(){let e=document.createElement(`style`);return e.setAttribute(ge,``),e}var fn=`@layer base, theme, unsafe;

@layer base {
  :host {
    --diffs-bg: #fff;
    --diffs-fg: #000;
    --diffs-font-fallback:
      'SF Mono', Monaco, Consolas, 'Ubuntu Mono', 'Liberation Mono',
      'Courier New', monospace;
    --diffs-header-font-fallback:
      system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue',
      'Noto Sans', 'Liberation Sans', Arial, sans-serif;

    --diffs-mixer: light-dark(black, white);
    --diffs-gap-fallback: 8px;

    --diffs-added-light: #0dbe4e;
    --diffs-added-dark: #5ecc71;
    --diffs-modified-light: #009fff;
    --diffs-modified-dark: #69b1ff;
    --diffs-deleted-light: #ff2e3f;
    --diffs-deleted-dark: #ff6762;

    /*
    // Available CSS Color Overrides
    --diffs-bg-buffer-override
    --diffs-bg-hover-override
    --diffs-bg-context-override
    --diffs-bg-separator-override

    --diffs-fg-number-override
    --diffs-fg-number-addition-override
    --diffs-fg-number-deletion-override
    --diffs-fg-conflict-marker-override

    --diffs-deletion-color-override
    --diffs-addition-color-override
    --diffs-modified-color-override

    --diffs-bg-deletion-override
    --diffs-bg-deletion-number-override
    --diffs-bg-deletion-hover-override
    --diffs-bg-deletion-emphasis-override

    --diffs-bg-addition-override
    --diffs-bg-addition-number-override
    --diffs-bg-addition-hover-override
    --diffs-bg-addition-emphasis-override

    // Line Selection Color Overrides (for enableLineSelection)
    --diffs-selection-color-override
    --diffs-bg-selection-override
    --diffs-bg-selection-number-override
    --diffs-bg-selection-background-override
    --diffs-bg-selection-number-background-override

    // Available CSS Layout Overrides
    --diffs-gap-inline
    --diffs-gap-block
    --diffs-gap-style
    --diffs-tab-size
  */

    color-scheme: light dark;
    display: block;
    font-family: var(
      --diffs-header-font-family,
      var(--diffs-header-font-fallback)
    );
    font-size: var(--diffs-font-size, 13px);
    line-height: var(--diffs-line-height, 20px);
    font-feature-settings: var(--diffs-font-features);
  }

  /* NOTE(mdo): Some semantic HTML elements (e.g. \`pre\`, \`code\`) have default
 * user-agent styles. These must be overridden to use our custom styles. */
  pre,
  code,
  [data-error-wrapper] {
    isolation: isolate;
    margin: 0;
    padding: 0;
    display: block;
    outline: none;
    font-family: var(--diffs-font-family, var(--diffs-font-fallback));
  }

  pre,
  code {
    background-color: var(--diffs-bg);
  }

  code {
    contain: content;
  }

  *,
  *::before,
  *::after {
    box-sizing: border-box;
  }

  [data-icon-sprite] {
    display: none;
  }

  /* NOTE(mdo): Headers and separators are within pre/code, so we need to reset
   * their font-family explicitly. */
  [data-diffs-header],
  [data-separator] {
    font-family: var(
      --diffs-header-font-family,
      var(--diffs-header-font-fallback)
    );
  }

  [data-file-info] {
    padding: 10px;
    font-weight: 700;
    color: var(--fg);
    /* NOTE(amadeus): we cannot use 'in oklch' because current versions of cursor
   * and vscode use an older build of chrome that appears to have a bug with
   * color-mix and 'in oklch', so use 'in lab' instead */
    background-color: color-mix(in lab, var(--bg) 98%, var(--fg));
    border-block: 1px solid color-mix(in lab, var(--bg) 95%, var(--fg));
  }

  [data-diffs-header],
  [data-diff],
  [data-file],
  [data-error-wrapper],
  [data-virtualizer-buffer] {
    --diffs-bg: light-dark(var(--diffs-light-bg), var(--diffs-dark-bg));
    /* NOTE(amadeus): we cannot use 'in oklch' because current versions of cursor
   * and vscode use an older build of chrome that appears to have a bug with
   * color-mix and 'in oklch', so use 'in lab' instead */
    --diffs-bg-buffer: var(
      --diffs-bg-buffer-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 92%, var(--diffs-mixer)),
        color-mix(in lab, var(--diffs-bg) 92%, var(--diffs-mixer))
      )
    );
    --diffs-bg-hover: var(
      --diffs-bg-hover-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 97%, var(--diffs-mixer)),
        color-mix(in lab, var(--diffs-bg) 91%, var(--diffs-mixer))
      )
    );

    --diffs-bg-context: var(
      --diffs-bg-context-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 98.5%, var(--diffs-mixer)),
        color-mix(in lab, var(--diffs-bg) 92.5%, var(--diffs-mixer))
      )
    );
    --diffs-bg-context-number: var(
      --diffs-bg-context-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg-context) 80%, var(--diffs-bg)),
        color-mix(in lab, var(--diffs-bg-context) 60%, var(--diffs-bg))
      )
    );
    --diffs-bg-conflict-marker: var(
      --diffs-bg-conflict-marker-override,
      light-dark(
        color-mix(
          in lab,
          var(--diffs-bg-context) 88%,
          var(--diffs-modified-base)
        ),
        color-mix(
          in lab,
          var(--diffs-bg-context) 80%,
          var(--diffs-modified-base)
        )
      )
    );
    --diffs-bg-conflict-current: var(
      --diffs-bg-conflict-current-override,
      light-dark(#e5f8ea, #274432)
    );
    --diffs-bg-conflict-base: var(
      --diffs-bg-conflict-base-override,
      light-dark(
        color-mix(
          in lab,
          var(--diffs-bg-context) 90%,
          var(--diffs-modified-base)
        ),
        color-mix(
          in lab,
          var(--diffs-bg-context) 82%,
          var(--diffs-modified-base)
        )
      )
    );
    --diffs-bg-conflict-incoming: var(
      --diffs-bg-conflict-incoming-override,
      light-dark(#e6f1ff, #253b5a)
    );
    --diffs-bg-conflict-marker-number: var(
      --diffs-bg-conflict-marker-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg-conflict-marker) 72%, var(--diffs-bg)),
        color-mix(in lab, var(--diffs-bg-conflict-marker) 54%, var(--diffs-bg))
      )
    );
    --diffs-bg-conflict-current-number: var(
      --diffs-bg-conflict-current-number-override,
      light-dark(#d7f1de, #30533d)
    );
    --diffs-bg-conflict-base-number: var(
      --diffs-bg-conflict-base-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg-conflict-base) 72%, var(--diffs-bg)),
        color-mix(in lab, var(--diffs-bg-conflict-base) 54%, var(--diffs-bg))
      )
    );
    --diffs-bg-conflict-incoming-number: var(
      --diffs-bg-conflict-incoming-number-override,
      light-dark(#d8e8ff, #2f4b73)
    );
    --conflict-bg-current: var(
      --conflict-bg-current-override,
      var(--diffs-bg-addition)
    );
    --conflict-bg-incoming: var(
      --conflict-bg-incoming-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 88%, var(--diffs-modified-base)),
        color-mix(in lab, var(--diffs-bg) 80%, var(--diffs-modified-base))
      )
    );
    --conflict-bg-current-number: var(
      --conflict-bg-current-number-override,
      var(--diffs-bg-addition-number)
    );
    --conflict-bg-incoming-number: var(
      --conflict-bg-incoming-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 91%, var(--diffs-modified-base)),
        color-mix(in lab, var(--diffs-bg) 85%, var(--diffs-modified-base))
      )
    );
    --conflict-bg-current-header: var(
      --conflict-bg-current-header-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 78%, var(--diffs-addition-base)),
        color-mix(in lab, var(--diffs-bg) 68%, var(--diffs-addition-base))
      )
    );
    --conflict-bg-incoming-header: var(
      --conflict-bg-incoming-header-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 78%, var(--diffs-modified-base)),
        color-mix(in lab, var(--diffs-bg) 68%, var(--diffs-modified-base))
      )
    );
    --conflict-bg-current-header-number: var(
      --conflict-bg-current-header-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 72%, var(--diffs-addition-base)),
        color-mix(in lab, var(--diffs-bg) 62%, var(--diffs-addition-base))
      )
    );
    --conflict-bg-incoming-header-number: var(
      --conflict-bg-incoming-header-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 72%, var(--diffs-modified-base)),
        color-mix(in lab, var(--diffs-bg) 62%, var(--diffs-modified-base))
      )
    );

    --diffs-bg-separator: var(
      --diffs-bg-separator-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 96%, var(--diffs-mixer)),
        color-mix(in lab, var(--diffs-bg) 85%, var(--diffs-mixer))
      )
    );

    --diffs-fg: light-dark(var(--diffs-light), var(--diffs-dark));
    --diffs-fg-number: var(
      --diffs-fg-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-fg) 65%, var(--diffs-bg)),
        color-mix(in lab, var(--diffs-fg) 65%, var(--diffs-bg))
      )
    );
    --diffs-fg-conflict-marker: var(
      --diffs-fg-conflict-marker-override,
      var(--diffs-fg-number)
    );

    --diffs-deletion-base: var(
      --diffs-deletion-color-override,
      light-dark(
        var(
          --diffs-light-deletion-color,
          var(--diffs-deletion-color, var(--diffs-deleted-light))
        ),
        var(
          --diffs-dark-deletion-color,
          var(--diffs-deletion-color, var(--diffs-deleted-dark))
        )
      )
    );
    --diffs-addition-base: var(
      --diffs-addition-color-override,
      light-dark(
        var(
          --diffs-light-addition-color,
          var(--diffs-addition-color, var(--diffs-added-light))
        ),
        var(
          --diffs-dark-addition-color,
          var(--diffs-addition-color, var(--diffs-added-dark))
        )
      )
    );
    --diffs-modified-base: var(
      --diffs-modified-color-override,
      light-dark(
        var(
          --diffs-light-modified-color,
          var(--diffs-modified-color, var(--diffs-modified-light))
        ),
        var(
          --diffs-dark-modified-color,
          var(--diffs-modified-color, var(--diffs-modified-dark))
        )
      )
    );

    /* NOTE(amadeus): we cannot use 'in oklch' because current versions of cursor
   * and vscode use an older build of chrome that appears to have a bug with
   * color-mix and 'in oklch', so use 'in lab' instead */
    --diffs-bg-deletion: var(
      --diffs-bg-deletion-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 88%, var(--diffs-deletion-base)),
        color-mix(in lab, var(--diffs-bg) 80%, var(--diffs-deletion-base))
      )
    );
    --diffs-bg-deletion-number: var(
      --diffs-bg-deletion-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 91%, var(--diffs-deletion-base)),
        color-mix(in lab, var(--diffs-bg) 85%, var(--diffs-deletion-base))
      )
    );
    --diffs-bg-deletion-hover: var(
      --diffs-bg-deletion-hover-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 80%, var(--diffs-deletion-base)),
        color-mix(in lab, var(--diffs-bg) 75%, var(--diffs-deletion-base))
      )
    );
    --diffs-bg-deletion-emphasis: var(
      --diffs-bg-deletion-emphasis-override,
      light-dark(
        rgb(from var(--diffs-deletion-base) r g b / 0.15),
        rgb(from var(--diffs-deletion-base) r g b / 0.2)
      )
    );

    --diffs-bg-addition: var(
      --diffs-bg-addition-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 88%, var(--diffs-addition-base)),
        color-mix(in lab, var(--diffs-bg) 80%, var(--diffs-addition-base))
      )
    );
    --diffs-bg-addition-number: var(
      --diffs-bg-addition-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 91%, var(--diffs-addition-base)),
        color-mix(in lab, var(--diffs-bg) 85%, var(--diffs-addition-base))
      )
    );
    --diffs-bg-addition-hover: var(
      --diffs-bg-addition-hover-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 80%, var(--diffs-addition-base)),
        color-mix(in lab, var(--diffs-bg) 70%, var(--diffs-addition-base))
      )
    );
    --diffs-bg-addition-emphasis: var(
      --diffs-bg-addition-emphasis-override,
      light-dark(
        rgb(from var(--diffs-addition-base) r g b / 0.15),
        rgb(from var(--diffs-addition-base) r g b / 0.2)
      )
    );

    --diffs-selection-base: var(--diffs-modified-base);
    --diffs-selection-number-fg: light-dark(
      color-mix(in lab, var(--diffs-selection-base) 65%, var(--diffs-mixer)),
      color-mix(in lab, var(--diffs-selection-base) 75%, var(--diffs-mixer))
    );
    --diffs-bg-selection: var(
      --diffs-bg-selection-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 82%, var(--diffs-selection-base)),
        color-mix(in lab, var(--diffs-bg) 75%, var(--diffs-selection-base))
      )
    );
    --diffs-bg-selection-number: var(
      --diffs-bg-selection-number-override,
      light-dark(
        color-mix(in lab, var(--diffs-bg) 75%, var(--diffs-selection-base)),
        color-mix(in lab, var(--diffs-bg) 60%, var(--diffs-selection-base))
      )
    );

    background-color: var(--diffs-bg);
    color: var(--diffs-fg);
  }

  [data-diff],
  [data-file] {
    /* This feels a bit crazy to me... so I need to think about it a bit more... */
    --diffs-grid-number-column-width: minmax(min-content, max-content);
    --diffs-code-grid: var(--diffs-grid-number-column-width) 1fr;

    &[data-dehydrated] {
      --diffs-code-grid: var(--diffs-grid-number-column-width) minmax(0, 1fr);
    }

    &[data-theme-type='light'],
    & {
      [data-line] span {
        color: light-dark(
          var(--diffs-token-light, var(--diffs-light)),
          var(--diffs-token-dark, var(--diffs-dark))
        );
        font-weight: var(--diffs-token-light-font-weight, inherit);
        font-style: var(--diffs-token-light-font-style, inherit);
        -webkit-text-decoration: var(--diffs-token-light-text-decoration, inherit);
                text-decoration: var(--diffs-token-light-text-decoration, inherit);
      }
    }

    &[data-theme-type='dark'] [data-line] span {
      font-weight: var(--diffs-token-dark-font-weight, inherit);
      font-style: var(--diffs-token-dark-font-style, inherit);
      -webkit-text-decoration: var(--diffs-token-dark-text-decoration, inherit);
              text-decoration: var(--diffs-token-dark-text-decoration, inherit);
    }

    &:hover [data-code]::-webkit-scrollbar-thumb {
      background-color: var(--diffs-bg-context);
    }
  }

  [data-line] span {
    background-color: light-dark(
      var(--diffs-token-light-bg, inherit),
      var(--diffs-token-dark-bg, inherit)
    );
  }

  [data-line],
  [data-gutter-buffer],
  [data-line-annotation],
  [data-no-newline] {
    color: var(--diffs-fg);
    background-color: var(--diffs-line-bg, var(--diffs-bg));
  }

  [data-no-newline] {
    -webkit-user-select: none;
            user-select: none;

    span {
      opacity: 0.6;
    }
  }

  @media (prefers-color-scheme: dark) {
    [data-diffs-header],
    [data-error-wrapper],
    [data-diff],
    [data-file] {
      color-scheme: dark;
    }

    [data-content] [data-line] span {
      font-weight: var(--diffs-token-dark-font-weight, inherit);
      font-style: var(--diffs-token-dark-font-style, inherit);
      -webkit-text-decoration: var(--diffs-token-dark-text-decoration, inherit);
              text-decoration: var(--diffs-token-dark-text-decoration, inherit);
    }
  }

  [data-diffs-header],
  [data-diff],
  [data-file] {
    &[data-theme-type='light'] {
      color-scheme: light;
    }

    &[data-theme-type='dark'] {
      color-scheme: dark;
    }
  }

  [data-diff-type='split'][data-overflow='scroll'] {
    display: grid;
    grid-template-columns: 1fr 1fr;

    [data-additions] {
      border-left: 1px solid var(--diffs-bg);
    }

    [data-deletions] {
      border-right: 1px solid var(--diffs-bg);
    }
  }

  [data-code] {
    display: grid;
    grid-auto-flow: dense;
    grid-template-columns: var(--diffs-code-grid);
    overflow: scroll clip;
    overscroll-behavior-x: none;
    tab-size: var(--diffs-tab-size, 2);
    align-self: flex-start;
    padding-top: var(--diffs-gap-block, var(--diffs-gap-fallback));
    padding-bottom: max(
      0px,
      calc(var(--diffs-gap-block, var(--diffs-gap-fallback)) - 6px)
    );
  }

  [data-container-size] {
    container-type: inline-size;
  }

  [data-code]::-webkit-scrollbar {
    width: 0;
    height: 6px;
  }

  [data-code]::-webkit-scrollbar-track {
    background: transparent;
  }

  [data-code]::-webkit-scrollbar-thumb {
    background-color: transparent;
    border: 1px solid transparent;
    background-clip: content-box;
    border-radius: 3px;
  }

  [data-code]::-webkit-scrollbar-corner {
    background-color: transparent;
  }

  /*
   * If we apply these rules globally it will mean that webkit will opt into the
   * standards compliant version of custom css scrollbars, which we do not want
   * because the custom stuff will look better
  */
  @supports (-moz-appearance: none) {
    [data-code] {
      scrollbar-width: thin;
      scrollbar-color: var(--diffs-bg-context) transparent;
      padding-bottom: var(--diffs-gap-block, var(--diffs-gap-fallback));
    }
  }

  [data-diffs-header] ~ [data-diff],
  [data-diffs-header] ~ [data-file] {
    [data-code],
    &[data-overflow='wrap'] {
      padding-top: 0;
    }
  }

  [data-gutter] {
    display: grid;
    grid-template-rows: subgrid;
    grid-template-columns: subgrid;
    grid-column: 1;
    z-index: 3;
    position: relative;
    background-color: var(--diffs-bg);

    [data-gutter-buffer],
    [data-column-number] {
      border-right: var(--diffs-gap-style, 2px solid var(--diffs-bg));
    }
  }

  [data-content] {
    display: grid;
    grid-template-rows: subgrid;
    grid-template-columns: subgrid;
    grid-column: 2;
    min-width: 0;
  }

  [data-diff-type='split'][data-overflow='wrap'] {
    display: grid;
    grid-auto-flow: dense;
    grid-template-columns: repeat(2, var(--diffs-code-grid));
    padding-block: var(--diffs-gap-block, var(--diffs-gap-fallback));

    [data-deletions] {
      display: contents;

      [data-gutter] {
        grid-column: 1;
      }

      [data-content] {
        grid-column: 2;
        border-right: 1px solid var(--diffs-bg);
      }
    }

    [data-additions] {
      display: contents;

      [data-gutter] {
        grid-column: 3;
        border-left: 1px solid var(--diffs-bg);
      }

      [data-content] {
        grid-column: 4;
      }
    }
  }

  [data-overflow='scroll'] [data-gutter] {
    position: sticky;
    left: 0;
  }

  [data-line-annotation][data-selected-line] {
    background-color: unset;

    &::before {
      content: '';
      /* FIXME(amadeus): This needs to be audited ... */
      position: sticky;
      top: 0;
      left: 0;
      display: block;
      border-right: var(--diffs-gap-style, 1px solid var(--diffs-bg));
      background-color: var(--diffs-bg-selection-number);
    }

    [data-annotation-content] {
      background-color: var(--diffs-bg-selection);
    }
  }

  [data-interactive-lines] [data-line] {
    cursor: pointer;
  }

  [data-content-buffer],
  [data-gutter-buffer] {
    position: relative;
    -webkit-user-select: none;
            user-select: none;
    min-height: 1lh;
  }

  [data-gutter-buffer='annotation'] {
    min-height: 0;
  }

  [data-gutter-buffer='buffer'] {
    background-size: 8px 8px;
    background-position: 0 0;
    background-origin: border-box;
    background-color: var(--diffs-bg);
    /* This is incredibley expensive... */
    background-image: repeating-linear-gradient(
      -45deg,
      transparent,
      transparent calc(3px * 1.414),
      rgb(from var(--diffs-bg-buffer) r g b / 0.8) calc(3px * 1.414),
      rgb(from var(--diffs-bg-buffer) r g b / 0.8) calc(4px * 1.414)
    );
  }

  [data-content-buffer] {
    grid-column: 1;
    /* We multiply by 1.414 (√2) to better approximate the diagonal repeat distance */
    background-size: 8px 8px;
    background-position: 5px 0;
    background-origin: border-box;
    background-color: var(--diffs-bg);
    /* This is incredibley expensive... */
    background-image: repeating-linear-gradient(
      -45deg,
      transparent,
      transparent calc(3px * 1.414),
      var(--diffs-bg-buffer) calc(3px * 1.414),
      var(--diffs-bg-buffer) calc(4px * 1.414)
    );
  }

  [data-separator] {
    box-sizing: content-box;
    background-color: var(--diffs-bg);
  }

  [data-separator='simple'] {
    min-height: 4px;
  }

  [data-separator='line-info'],
  [data-separator='line-info-basic'],
  [data-separator='metadata'],
  [data-separator='simple'] {
    background-color: var(--diffs-bg-separator);
  }

  [data-separator='line-info'],
  [data-separator='line-info-basic'],
  [data-separator='metadata'] {
    height: 32px;
    position: relative;
  }

  [data-separator-wrapper] {
    -webkit-user-select: none;
            user-select: none;
    fill: currentColor;
    position: absolute;
    inset-inline: 0;
    display: flex;
    align-items: center;
    background-color: var(--diffs-bg);
    height: 100%;
  }

  [data-content] [data-separator-wrapper] {
    display: none;
  }

  [data-separator='metadata'] [data-separator-wrapper] {
    inset-inline: 100% auto;
    padding-inline: 1ch;
    height: 100%;
    background-color: var(--diffs-bg-separator);
    color: var(--diffs-fg-number);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: min-content;
  }

  [data-separator='line-info'] {
    margin-block: var(--diffs-gap-block, var(--diffs-gap-fallback));
  }

  [data-separator='line-info-basic'],
  [data-separator='metadata'] {
    margin-block: 0;
  }

  [data-separator='line-info'][data-separator-first] {
    margin-top: 0;
  }

  [data-separator='line-info'][data-separator-last] {
    margin-bottom: 0;
  }

  [data-expand-index] [data-separator-wrapper] {
    display: grid;
    grid-template-columns: 32px auto;
  }

  [data-expand-index] [data-separator-wrapper][data-separator-multi-button] {
    grid-template-columns: 32px 32px auto;
  }

  [data-expand-button],
  [data-separator-content] {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    background-color: var(--diffs-bg-separator);
  }

  [data-expand-button] {
    justify-content: center;
    flex-shrink: 0;
    cursor: pointer;
    min-width: 32px;
    align-self: stretch;
    color: var(--diffs-fg-number);
    border-right: 2px solid var(--diffs-bg);

    &:hover {
      color: var(--diffs-fg);
    }
  }

  [data-expand-down] [data-icon] {
    transform: scaleY(-1);
  }

  [data-separator-content] {
    flex: 1 1 auto;
    padding: 0 1ch;
    height: 100%;
    color: var(--diffs-fg-number);

    overflow: hidden;
    justify-content: flex-start;
  }

  [data-separator='line-info'],
  [data-separator='line-info-basic'] {
    [data-separator-content] {
      height: 100%;
      -webkit-user-select: none;
              user-select: none;
      overflow: clip;
    }
  }

  @supports (width: 1cqi) {
    [data-unified] {
      [data-separator='line-info'] [data-separator-wrapper] {
        padding-inline: var(--diffs-gap-inline, var(--diffs-gap-fallback));
        width: 100cqi;

        [data-separator-content] {
          border-radius: 6px;
        }
      }

      [data-separator='line-info'][data-expand-index]
        [data-separator-wrapper]
        [data-separator-content] {
        border-top-left-radius: unset;
        border-bottom-left-radius: unset;
      }
    }

    [data-gutter] {
      [data-separator='line-info'] [data-separator-wrapper] {
        padding-left: var(--diffs-gap-inline, var(--diffs-gap-fallback));
      }

      [data-separator='line-info'] [data-separator-content] {
        border-top-left-radius: 6px;
        border-bottom-left-radius: 6px;
      }

      [data-separator='line-info'][data-expand-index] [data-separator-content] {
        border-top-left-radius: unset;
        border-bottom-left-radius: unset;
      }
    }

    [data-additions] {
      [data-content] [data-separator='line-info'] {
        background-color: var(--diffs-bg);

        [data-separator-wrapper] {
          display: none;
        }
      }

      [data-gutter] [data-separator='line-info'] [data-separator-wrapper] {
        display: block;
        height: 100%;
        background-color: var(--diffs-bg-separator);
        border-top-right-radius: 6px;
        border-bottom-right-radius: 6px;

        [data-separator-content],
        [data-expand-button] {
          display: none;
        }
      }
    }

    [data-overflow='scroll']
      [data-additions]
      [data-gutter]
      [data-separator='line-info']
      [data-separator-wrapper] {
      width: calc(100cqi - var(--diffs-gap-inline, var(--diffs-gap-fallback)));
    }

    [data-overflow='wrap']
      [data-additions]
      [data-content]
      [data-separator='line-info']
      [data-separator-wrapper] {
      background-color: var(--diffs-bg-separator);
      display: block;
      height: 100%;
      margin-right: var(--diffs-gap-inline, var(--diffs-gap-fallback));
      border-top-right-radius: 6px;
      border-bottom-right-radius: 6px;

      [data-separator-content],
      [data-expand-button] {
        display: none;
      }
    }

    [data-separator='line-info'] [data-separator-wrapper] {
      [data-expand-both],
      [data-expand-down],
      [data-expand-up] {
        border-top-left-radius: 6px;
        border-bottom-left-radius: 6px;
      }
    }

    @media (pointer: fine) {
      [data-separator='line-info'] [data-separator-wrapper] {
        &[data-separator-multi-button] {
          [data-expand-up] {
            border-top-left-radius: 6px;
            border-bottom-left-radius: unset;
          }

          [data-expand-down] {
            border-bottom-left-radius: 6px;
            border-top-left-radius: unset;
          }
        }
      }
    }
  }

  @media (pointer: coarse) {
    [data-separator='line-info-basic']
      [data-separator-wrapper][data-separator-multi-button] {
      grid-template-columns: 34px 34px auto;

      [data-separator-content] {
        grid-column: unset;
        grid-row: unset;
      }
    }

    @supports (width: 1cqi) {
      [data-separator='line-info'] [data-separator-wrapper] {
        [data-expand-both],
        [data-expand-down],
        [data-expand-up] {
          border-top-left-radius: 6px;
          border-bottom-left-radius: 6px;
        }

        &[data-separator-multi-button] {
          [data-expand-up] {
            border-top-left-radius: 6px;
            border-bottom-left-radius: 6px;
          }

          [data-expand-down] {
            border-bottom-left-radius: unset;
            border-top-left-radius: unset;
          }
        }
      }
    }
  }

  @media (pointer: fine) {
    [data-separator-wrapper][data-separator-multi-button] {
      display: grid;
      grid-template-rows: 50% 50%;

      [data-separator-content] {
        grid-column: 2;
        grid-row: 1 / -1;
        min-width: min-content;
      }

      [data-expand-button] {
        grid-column: 1;
      }
    }

    [data-separator='line-info'] [data-separator-wrapper],
    [data-separator='line-info']
      [data-separator-wrapper][data-separator-multi-button] {
      grid-template-columns: 34px auto;
    }

    [data-separator='line-info-basic'][data-expand-index]
      [data-separator-wrapper] {
      grid-template-columns: 100% auto;
    }

    [data-separator='line-info'],
    [data-separator='line-info-basic'] {
      [data-separator-multi-button] {
        [data-expand-up] {
          border-bottom: 1px solid var(--diffs-bg);
          border-right: 2px solid var(--diffs-bg);
        }
        [data-expand-down] {
          border-top: 1px solid var(--diffs-bg);
          border-right: 2px solid var(--diffs-bg);
        }
      }
    }
  }

  [data-additions] [data-gutter] [data-separator-wrapper],
  [data-additions] [data-separator='line-info-basic'] [data-separator-wrapper],
  [data-content] [data-separator-wrapper] {
    display: none;
  }

  [data-line-annotation],
  [data-gutter-buffer='annotation'] {
    --diffs-line-bg: var(--diffs-bg-context);
  }

  [data-merge-conflict-actions],
  [data-gutter-buffer='merge-conflict-action'] {
    --diffs-line-bg: var(--diffs-bg-context);
  }

  [data-has-merge-conflict] [data-line-annotation],
  [data-has-merge-conflict] [data-gutter-buffer='annotation'] {
    --diffs-line-bg: var(--diffs-bg);
  }

  [data-has-merge-conflict] [data-gutter-buffer='merge-conflict-action'] {
    --diffs-line-bg: var(--diffs-bg);
  }

  [data-line-annotation] {
    min-height: var(--diffs-annotation-min-height, 0);
    z-index: 2;
  }

  [data-merge-conflict-actions] {
    z-index: 2;
  }

  [data-separator='custom'] {
    display: grid;
    grid-template-columns: subgrid;
  }

  [data-line],
  [data-column-number],
  [data-no-newline] {
    position: relative;
    padding-inline: 1ch;
  }

  [data-indicators='classic'] [data-line] {
    padding-inline-start: 2ch;
  }

  [data-indicators='classic'] {
    [data-line-type='change-addition'],
    [data-line-type='change-deletion'] {
      &[data-no-newline],
      &[data-line] {
        &::before {
          display: inline-block;
          width: 1ch;
          height: 1lh;
          position: absolute;
          top: 0;
          left: 0;
          -webkit-user-select: none;
                  user-select: none;
        }
      }
    }

    [data-line-type='change-addition'] {
      &[data-line],
      &[data-no-newline] {
        &::before {
          content: '+';
          color: var(--diffs-addition-base);
        }
      }
    }

    [data-line-type='change-deletion'] {
      &[data-line],
      &[data-no-newline] {
        &::before {
          content: '-';
          color: var(--diffs-deletion-base);
        }
      }
    }
  }

  [data-indicators='bars'] {
    [data-line-type='change-deletion'],
    [data-line-type='change-addition'] {
      &[data-column-number] {
        &::before {
          content: '';
          display: block;
          width: 4px;
          height: 100%;
          position: absolute;
          top: 0;
          left: 0;
          -webkit-user-select: none;
                  user-select: none;
          contain: strict;
        }
      }
    }

    [data-line-type='change-deletion'] {
      &[data-column-number] {
        &::before {
          background-image: linear-gradient(
            0deg,
            var(--diffs-bg-deletion) 50%,
            var(--diffs-deletion-base) 50%
          );
          background-repeat: repeat;
          background-size: 2px 2px;
          background-size: calc(1lh / round(1lh / 2px))
            calc(1lh / round(1lh / 2px));
        }
      }
    }

    [data-line-type='change-addition'] {
      &[data-column-number] {
        &::before {
          background-color: var(--diffs-addition-base);
        }
      }
    }
  }

  [data-overflow='wrap'] {
    [data-line],
    [data-annotation-content] {
      white-space: pre-wrap;
      word-break: break-word;
    }
  }

  [data-overflow='scroll'] [data-line] {
    white-space: pre;
    min-height: 1lh;
  }

  [data-column-number] {
    box-sizing: content-box;
    text-align: right;
    -webkit-user-select: none;
            user-select: none;
    background-color: var(--diffs-bg);
    color: var(--diffs-fg-number);
    padding-left: 2ch;
  }

  [data-line-number-content] {
    display: inline-block;
    min-width: var(
      --diffs-min-number-column-width,
      var(--diffs-min-number-column-width-default, 3ch)
    );
  }

  [data-disable-line-numbers] {
    [data-column-number] {
      min-width: 4px;
      padding: 0;
    }

    [data-line-number-content] {
      display: none;
    }

    [data-gutter-utility-slot] {
      right: unset;
      left: 0;
      justify-content: flex-start;
    }

    &[data-indicators='bars'] [data-gutter-utility-slot] {
      /* Using 5px here because theres a 1px separator after the bar */
      left: 5px;
    }
  }

  [data-file][data-disable-line-numbers] {
    [data-gutter-buffer],
    [data-column-number] {
      min-width: 0;
      border-right: 0;
    }
  }

  [data-interactive-line-numbers] [data-column-number] {
    cursor: pointer;
  }

  [data-diff-span] {
    border-radius: 3px;
    -webkit-box-decoration-break: clone;
            box-decoration-break: clone;
  }

  [data-line-type='change-addition'] {
    &[data-column-number] {
      color: var(
        --diffs-fg-number-addition-override,
        var(--diffs-addition-base)
      );
    }

    > [data-diff-span] {
      background-color: var(--diffs-bg-addition-emphasis);
    }
  }

  [data-line-type='change-deletion'] {
    &[data-column-number] {
      color: var(
        --diffs-fg-number-deletion-override,
        var(--diffs-deletion-base)
      );
    }

    [data-diff-span] {
      background-color: var(--diffs-bg-deletion-emphasis);
    }
  }

  [data-background] [data-line-type='change-addition'] {
    --diffs-line-bg: var(--diffs-bg-addition);

    &[data-column-number] {
      background-color: var(--diffs-bg-addition-number);
    }
  }

  [data-background] [data-line-type='change-deletion'] {
    --diffs-line-bg: var(--diffs-bg-deletion);

    &[data-column-number] {
      background-color: var(--diffs-bg-deletion-number);
    }
  }

  [data-merge-conflict^='marker-'][data-line] {
    &[data-line-type='context'],
    &[data-line-type='context-expanded'] {
      color: var(--diffs-fg);

      span {
        color: var(--diffs-fg) !important;
      }
    }
  }

  [data-merge-conflict='marker-start'][data-line] {
    &[data-line-type='context'],
    &[data-line-type='context-expanded'] {
      &::after {
        content: '  (Current Change)';
        color: var(--diffs-fg-conflict-marker);
        opacity: 1;
        font-style: normal;
        font-family: var(
          --diffs-header-font-family,
          var(--diffs-header-font-fallback)
        );
      }
    }
  }

  [data-merge-conflict='marker-end'][data-line] {
    &[data-line-type='context'],
    &[data-line-type='context-expanded'] {
      &::after {
        content: '  (Incoming Change)';
        color: var(--diffs-fg-conflict-marker);
        opacity: 1;
        font-style: normal;
        font-family: var(
          --diffs-header-font-family,
          var(--diffs-header-font-fallback)
        );
      }
    }
  }

  [data-merge-conflict='marker-start'],
  [data-merge-conflict='marker-base'],
  [data-merge-conflict='marker-separator'],
  [data-merge-conflict='marker-end'] {
    &[data-line],
    &[data-no-newline] {
      background-color: var(--diffs-bg-conflict-marker);
    }

    &[data-column-number] {
      background-color: var(--diffs-bg-conflict-marker-number);
      color: var(--diffs-fg-conflict-marker);

      [data-line-number-content] {
        color: var(--diffs-fg-conflict-marker);
      }
    }
  }

  [data-merge-conflict='current'] {
    &[data-line],
    &[data-no-newline] {
      background-color: var(--conflict-bg-current);
    }

    &[data-column-number] {
      background-color: var(--conflict-bg-current-number);
      color: var(--diffs-addition-base);
    }
  }

  [data-merge-conflict='marker-start'] {
    &[data-line],
    &[data-no-newline] {
      background-color: var(--conflict-bg-current-header);
    }

    &[data-column-number] {
      background-color: var(--conflict-bg-current-header-number);
      color: var(--diffs-addition-base);

      [data-line-number-content] {
        color: var(--diffs-addition-base);
      }
    }
  }

  [data-merge-conflict='marker-end'] {
    &[data-line],
    &[data-no-newline] {
      background-color: var(--conflict-bg-incoming-header);
    }

    &[data-column-number] {
      background-color: var(--conflict-bg-incoming-header-number);
      color: var(--diffs-modified-base);

      [data-line-number-content] {
        color: var(--diffs-modified-base);
      }
    }
  }

  [data-merge-conflict='marker-separator'] {
    &[data-line],
    &[data-no-newline] {
      background-color: var(--diffs-bg);
    }

    &[data-column-number] {
      background-color: var(--diffs-bg);
    }
  }

  [data-merge-conflict='base'] {
    &[data-line],
    &[data-no-newline] {
      background-color: var(--diffs-bg-conflict-base);
    }

    &[data-column-number] {
      background-color: var(--diffs-bg-conflict-base-number);
      color: var(--diffs-modified-base);
    }
  }

  [data-merge-conflict='incoming'] {
    &[data-line],
    &[data-no-newline] {
      background-color: var(--conflict-bg-incoming);
    }

    &[data-column-number] {
      background-color: var(--conflict-bg-incoming-number);
      color: var(--diffs-modified-base);
    }
  }

  @media (pointer: fine) {
    [data-column-number],
    [data-line] {
      &[data-hovered] {
        background-color: var(--diffs-bg-hover);
      }
    }

    [data-background] {
      [data-column-number],
      [data-line] {
        &[data-hovered] {
          &[data-line-type='change-deletion'] {
            background-color: var(--diffs-bg-deletion-hover);
          }

          &[data-line-type='change-addition'] {
            background-color: var(--diffs-bg-addition-hover);
          }
        }
      }
    }
  }

  [data-diffs-header] {
    position: relative;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: center;
    gap: var(--diffs-gap-inline, var(--diffs-gap-fallback));
    min-height: calc(
      1lh + (var(--diffs-gap-block, var(--diffs-gap-fallback)) * 3)
    );
    padding-inline: 16px;
    top: 0;
    z-index: 2;
  }

  [data-header-content] {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: var(--diffs-gap-inline, var(--diffs-gap-fallback));
    min-width: 0;
    white-space: nowrap;
  }

  [data-header-content] [data-prev-name],
  [data-header-content] [data-title] {
    direction: rtl;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
    white-space: nowrap;
  }

  [data-prev-name] {
    opacity: 0.7;
  }

  [data-rename-icon] {
    fill: currentColor;
    flex-shrink: 0;
    flex-grow: 0;
  }

  [data-diffs-header] [data-metadata] {
    display: flex;
    align-items: center;
    gap: 1ch;
    white-space: nowrap;
  }

  [data-diffs-header] [data-additions-count] {
    font-family: var(--diffs-font-family, var(--diffs-font-fallback));
    color: var(--diffs-addition-base);
  }

  [data-diffs-header] [data-deletions-count] {
    font-family: var(--diffs-font-family, var(--diffs-font-fallback));
    color: var(--diffs-deletion-base);
  }

  [data-annotation-content] {
    position: relative;
    display: flow-root;
    align-self: flex-start;
    z-index: 2;
    min-width: 0;
    isolation: isolate;
  }

  [data-merge-conflict-actions-content] {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding-inline: 0.5rem;
    min-height: 1.75rem;
    font-family: var(
      --diffs-header-font-family,
      var(--diffs-header-font-fallback)
    );
    font-size: 0.75rem;
    line-height: 1.2;
    color: var(--diffs-fg);
  }

  [data-merge-conflict-action] {
    appearance: none;
    border: 0;
    background: transparent;
    color: var(--diffs-fg-number);
    font: inherit;
    font-style: normal;
    cursor: pointer;
    padding: 0;
  }

  [data-merge-conflict-action]:hover {
    color: var(--diffs-fg);
  }

  [data-merge-conflict-action='current']:hover {
    color: var(--diffs-addition-base);
  }

  [data-merge-conflict-action='incoming']:hover {
    color: var(--diffs-modified-base);
  }

  [data-merge-conflict-action-separator] {
    color: var(--diffs-fg-number);
    opacity: 0.6;
    -webkit-user-select: none;
            user-select: none;
  }

  /* Sticky positioning has a composite costs, so we should _only_ pay it if we
   * need to */
  [data-overflow='scroll'] [data-annotation-content] {
    position: sticky;
    width: var(--diffs-column-content-width, auto);
    left: var(--diffs-column-number-width, 0);
  }

  [data-overflow='scroll'] [data-merge-conflict-actions-content] {
    position: sticky;
    width: var(--diffs-column-content-width, auto);
    left: var(--diffs-column-number-width, 0);
  }

  /* Undo some of the stuff that the 'pre' tag does */
  [data-annotation-slot] {
    text-wrap-mode: wrap;
    word-break: normal;
    white-space-collapse: collapse;
  }

  [data-change-icon] {
    fill: currentColor;
    flex-shrink: 0;
  }

  [data-change-icon='change'],
  [data-change-icon='rename-pure'],
  [data-change-icon='rename-changed'] {
    color: var(--diffs-modified-base);
  }

  [data-change-icon='new'] {
    color: var(--diffs-addition-base);
  }

  [data-change-icon='deleted'] {
    color: var(--diffs-deletion-base);
  }

  [data-change-icon='file'] {
    opacity: 0.6;
  }

  /* Line selection highlighting */
  [data-selected-line] {
    &[data-gutter-buffer='annotation'],
    &[data-column-number] {
      color: var(--diffs-selection-number-fg);
      background-color: var(--diffs-bg-selection-number);
    }

    &[data-line] {
      background-color: var(--diffs-bg-selection);
    }
  }

  [data-line-type='change-addition'],
  [data-line-type='change-deletion'] {
    &[data-selected-line] {
      &[data-line],
      &[data-line][data-hovered] {
        background-color: light-dark(
          color-mix(
            in lab,
            var(--diffs-line-bg, var(--diffs-bg)) 82%,
            var(--diffs-selection-base)
          ),
          color-mix(
            in lab,
            var(--diffs-line-bg, var(--diffs-bg)) 75%,
            var(--diffs-selection-base)
          )
        );
      }

      &[data-column-number],
      &[data-column-number][data-hovered] {
        color: var(--diffs-selection-number-fg);
        background-color: light-dark(
          color-mix(
            in lab,
            var(--diffs-line-bg, var(--diffs-bg)) 75%,
            var(--diffs-selection-base)
          ),
          color-mix(
            in lab,
            var(--diffs-line-bg, var(--diffs-bg)) 60%,
            var(--diffs-selection-base)
          )
        );
      }
    }
  }

  [data-gutter-utility-slot] {
    position: absolute;
    top: 0;
    bottom: 0;
    right: 0;
    display: flex;
    justify-content: flex-end;
  }

  [data-unmodified-lines] {
    display: block;
    overflow: hidden;
    min-width: 0;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 0 1 auto;
  }

  [data-error-wrapper] {
    overflow: auto;
    padding: var(--diffs-gap-block, var(--diffs-gap-fallback))
      var(--diffs-gap-inline, var(--diffs-gap-fallback));
    max-height: 400px;
    scrollbar-width: none;

    [data-error-message] {
      font-weight: bold;
      font-size: 18px;
      color: var(--diffs-deletion-base);
    }

    [data-error-stack] {
      color: var(--diffs-fg-number);
    }
  }

  [data-placeholder] {
    contain: strict;
  }

  [data-utility-button] {
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    appearance: none;
    width: 1lh;
    height: 1lh;
    margin-right: calc((1lh - 1ch) * -1);
    padding: 0;
    cursor: pointer;
    font-size: var(--diffs-font-size, 13px);
    line-height: var(--diffs-line-height, 20px);
    border-radius: 4px;
    background-color: var(--diffs-modified-base);
    color: var(--diffs-bg);
    fill: currentColor;
    position: relative;
    z-index: 4;
  }
}
`,pn=`@layer base, theme, unsafe;`;function mn(e){return`${pn}
@layer unsafe {
  ${e}
}`}function hn({code:e,pre:t,columnType:n,rowSpan:r,containerSize:i=!1}={}){return e??(e=document.createElement(`code`),e.setAttribute(`data-code`,``),n!=null&&e.setAttribute(`data-${n}`,``),t?.appendChild(e)),r==null?e.style.removeProperty(`grid-row`):e.style.setProperty(`grid-row`,`span ${r}`),i?e.setAttribute(`data-container-size`,``):e.removeAttribute(`data-container-size`),e}function gn(e,t){if(t==null)return;let n=e.shadowRoot??e.attachShadow({mode:`open`});n.innerHTML===``&&(n.innerHTML=t)}function _n(e,{type:t,diffIndicators:n,disableBackground:r,disableLineNumbers:i,overflow:a,split:o,themeStyles:s,themeType:c,totalLines:l,customProperties:u}){if(u!=null)for(let t in u){let n=u[t];n!=null&&e.setAttribute(t,`${n}`)}switch(t===`diff`?(e.setAttribute(`data-diff`,``),e.removeAttribute(`data-file`)):(e.setAttribute(`data-file`,``),e.removeAttribute(`data-diff`)),c===`system`?e.removeAttribute(`data-theme-type`):e.setAttribute(`data-theme-type`,c),n){case`bars`:case`classic`:e.setAttribute(`data-indicators`,n);break;case`none`:e.removeAttribute(`data-indicators`);break}return i?e.setAttribute(`data-disable-line-numbers`,``):e.removeAttribute(`data-disable-line-numbers`),r?e.removeAttribute(`data-background`):e.setAttribute(`data-background`,``),t===`diff`?e.setAttribute(`data-diff-type`,o?`split`:`single`):e.removeAttribute(`data-diff-type`),e.setAttribute(`data-overflow`,a),e.tabIndex=0,e.style=s,e.style.setProperty(`--diffs-min-number-column-width-default`,`${`${l}`.length}ch`),e}if(typeof HTMLElement<`u`&&customElements.get(`diffs-container`)==null){let e;class t extends HTMLElement{constructor(){if(super(),this.shadowRoot!=null)return;let t=this.attachShadow({mode:`open`});e??(e=new CSSStyleSheet,e.replaceSync(fn)),t.adoptedStyleSheets=[e]}}customElements.define(Ie,t)}var vn=class{isDeletionsScrolling=!1;isAdditionsScrolling=!1;timeoutId=-1;codeDeletions;codeAdditions;enabled=!1;cleanUp(){this.enabled&&=(this.codeDeletions?.removeEventListener(`scroll`,this.handleDeletionsScroll),this.codeAdditions?.removeEventListener(`scroll`,this.handleAdditionsScroll),clearTimeout(this.timeoutId),this.codeDeletions=void 0,this.codeAdditions=void 0,!1)}setup(e,t,n){if(t==null||n==null)for(let r of e.children??[])r instanceof HTMLElement&&(`deletions`in r.dataset?t=r:`additions`in r.dataset&&(n=r));if(n==null||t==null){this.cleanUp();return}this.codeDeletions!==t&&(this.codeDeletions?.removeEventListener(`scroll`,this.handleDeletionsScroll),this.codeDeletions=t,t.addEventListener(`scroll`,this.handleDeletionsScroll,{passive:!0})),this.codeAdditions!==n&&(this.codeAdditions?.removeEventListener(`scroll`,this.handleAdditionsScroll),this.codeAdditions=n,n.addEventListener(`scroll`,this.handleAdditionsScroll,{passive:!0})),this.enabled=!0}handleDeletionsScroll=()=>{this.isAdditionsScrolling||(this.isDeletionsScrolling=!0,clearTimeout(this.timeoutId),this.timeoutId=setTimeout(()=>{this.isDeletionsScrolling=!1},300),this.codeAdditions?.scrollTo({left:this.codeDeletions?.scrollLeft}))};handleAdditionsScroll=()=>{this.isDeletionsScrolling||(this.isAdditionsScrolling=!0,clearTimeout(this.timeoutId),this.timeoutId=setTimeout(()=>{this.isAdditionsScrolling=!1},300),this.codeDeletions?.scrollTo({left:this.codeAdditions?.scrollLeft}))}};function yn(e){return W({tagName:`div`,properties:{"data-content-buffer":``,"data-buffer-size":e,style:`grid-row: span ${e};min-height:calc(${e} * 1lh)`}})}function bn(e){return W({tagName:`div`,children:[W({tagName:`span`,children:[je(`No newline at end of file`)]})],properties:{"data-no-newline":``,"data-line-type":e,"data-column-content":``}})}function xn(e){return W({tagName:`div`,children:[ke({name:e===`both`?`diffs-icon-expand-all`:`diffs-icon-expand`,properties:{"data-icon":``}})],properties:{"data-expand-button":``,"data-expand-both":e===`both`?``:void 0,"data-expand-up":e===`up`?``:void 0,"data-expand-down":e===`down`?``:void 0}})}function Sn({type:e,content:t,expandIndex:n,chunked:r=!1,slotName:i,isFirstHunk:a,isLastHunk:o}){let s=[];if(e===`metadata`&&t!=null&&s.push(W({tagName:`div`,children:[je(t)],properties:{"data-separator-wrapper":``}})),(e===`line-info`||e===`line-info-basic`)&&t!=null){let e=[];n!=null&&(r?(a||e.push(xn(`up`)),o||e.push(xn(`down`))):e.push(xn(!a&&!o?`both`:a?`down`:`up`))),e.push(W({tagName:`div`,children:[W({tagName:`span`,children:[je(t)],properties:{"data-unmodified-lines":``}})],properties:{"data-separator-content":``}})),s.push(W({tagName:`div`,children:e,properties:{"data-separator-wrapper":``,"data-separator-multi-button":e.length>2?``:void 0}}))}return e===`custom`&&i!=null&&s.push(W({tagName:`slot`,properties:{name:i}})),W({tagName:`div`,children:s,properties:{"data-separator":s.length===0?`simple`:e,"data-expand-index":n,"data-separator-first":a?``:void 0,"data-separator-last":o?``:void 0}})}function Cn(e,t){return`hunk-separator-${e}-${t}`}function wn(e){let t=e.at(-1);return t==null?0:Math.max(t.additionStart+t.additionCount,t.deletionStart+t.deletionCount)}function Tn(e){return e.startingLine===0&&e.totalLines===1/0&&e.bufferBefore===0&&e.bufferAfter===0}var En=-1,Dn=class{__id=`diff-hunks-renderer:${++En}`;highlighter;diff;expandedHunks=new Map;deletionAnnotations={};additionAnnotations={};computedLang=`text`;renderCache;constructor(e={theme:st},t,n){this.options=e,this.onRenderUpdate=t,this.workerManager=n,n?.isWorkingPool()!==!0&&(this.highlighter=Kt(e.theme??st)?ye():void 0)}cleanUp(){this.highlighter=void 0,this.diff=void 0,this.renderCache=void 0,this.workerManager?.cleanUpPendingTasks(this),this.workerManager=void 0,this.onRenderUpdate=void 0}recycle(){this.highlighter=void 0,this.diff=void 0,this.renderCache=void 0,this.workerManager?.cleanUpPendingTasks(this)}setOptions(e){this.options=e}mergeOptions(e){this.options={...this.options,...e}}setThemeType(e){this.getOptionsWithDefaults().themeType!==e&&this.mergeOptions({themeType:e})}expandHunk(e,t,n=this.getOptionsWithDefaults().expansionLineCount){let r={...this.expandedHunks.get(e)??{fromStart:0,fromEnd:0}};(t===`up`||t===`both`)&&(r.fromStart+=n),(t===`down`||t===`both`)&&(r.fromEnd+=n),this.renderCache?.highlighted!==!0&&(this.renderCache=void 0),this.expandedHunks.set(e,r)}getExpandedHunk(e){return this.expandedHunks.get(e)??Oe}getExpandedHunksMap(){return this.expandedHunks}setLineAnnotations(e){this.additionAnnotations={},this.deletionAnnotations={};for(let t of e){let e=(()=>{switch(t.side){case`deletions`:return this.deletionAnnotations;case`additions`:return this.additionAnnotations}})(),n=e[t.lineNumber]??[];e[t.lineNumber]=n,n.push(t)}}getUnifiedLineDecoration({lineType:e}){return{gutterLineType:e}}getSplitLineDecoration({side:e,type:t}){return t===`change`?{gutterLineType:e===`deletions`?`change-deletion`:`change-addition`}:{gutterLineType:t}}createAnnotationElement(e){return Jt(e)}getOptionsWithDefaults(){let{diffIndicators:e=`bars`,diffStyle:t=`split`,disableBackground:n=!1,disableFileHeader:r=!1,disableLineNumbers:i=!1,disableVirtualizationBuffers:a=!1,collapsed:o=!1,expandUnchanged:s=!1,collapsedContextThreshold:c=1,expansionLineCount:l=100,hunkSeparators:u=`line-info`,lineDiffType:d=`word-alt`,maxLineDiffLength:f=1e3,overflow:p=`scroll`,theme:m=st,themeType:h=`system`,tokenizeMaxLineLength:g=1e3,useCSSClasses:_=!1}=this.options;return{diffIndicators:e,diffStyle:t,disableBackground:n,disableFileHeader:r,disableLineNumbers:i,disableVirtualizationBuffers:a,collapsed:o,expandUnchanged:s,collapsedContextThreshold:c,expansionLineCount:l,hunkSeparators:u,lineDiffType:d,maxLineDiffLength:f,overflow:p,theme:this.workerManager?.getDiffRenderOptions().theme??m,themeType:h,tokenizeMaxLineLength:g,useCSSClasses:_}}async initializeHighlighter(){return this.highlighter=await it(tn(this.computedLang,this.options)),this.highlighter}hydrate(e){if(e==null)return;this.diff=e;let{options:t}=this.getRenderOptions(e),n=this.workerManager?.getDiffResultCache(e);n!=null&&!On(t,n.options)&&(n=void 0),this.renderCache??={diff:e,highlighted:!0,options:t,result:n?.result,renderRange:void 0},this.workerManager?.isWorkingPool()===!0&&this.renderCache.result==null?this.workerManager.highlightDiffAST(this,this.diff):this.asyncHighlight(e).then(({result:t,options:n})=>{this.onHighlightSuccess(e,t,n)})}getRenderOptions(e){let t=(()=>{if(this.workerManager?.isWorkingPool()===!0)return this.workerManager.getDiffRenderOptions();let{theme:e,tokenizeMaxLineLength:t,lineDiffType:n}=this.getOptionsWithDefaults();return{theme:e,tokenizeMaxLineLength:t,lineDiffType:n}})();this.getOptionsWithDefaults();let{renderCache:n}=this;return n?.result==null||e!==n.diff||!On(t,n.options)?{options:t,forceRender:!0}:{options:t,forceRender:!1}}renderDiff(e=this.renderCache?.diff,t=Ne){if(e==null)return;let{expandUnchanged:n=!1,collapsedContextThreshold:r}=this.getOptionsWithDefaults(),i=this.workerManager?.getDiffResultCache(e);i!=null&&this.renderCache==null&&(this.renderCache={diff:e,highlighted:!0,renderRange:void 0,...i});let{options:a,forceRender:o}=this.getRenderOptions(e);if(this.renderCache??={diff:e,highlighted:!1,options:a,result:void 0,renderRange:void 0},this.workerManager?.isWorkingPool()===!0)(this.renderCache.result==null||!this.renderCache.highlighted&&!qt(this.renderCache.renderRange,t))&&(this.renderCache.result=this.workerManager.getPlainDiffAST(e,t.startingLine,t.totalLines,Tn(t)||n?!0:this.expandedHunks,r),this.renderCache.renderRange=t),t.totalLines>0&&(!this.renderCache.highlighted||o)&&this.workerManager.highlightDiffAST(this,e);else{this.computedLang=e.lang??Ue(e.name);let t=this.highlighter!=null&&Kt(a.theme),n=this.highlighter!=null&&Gt(this.computedLang);if(this.highlighter!=null&&t&&(o||!this.renderCache.highlighted&&n||this.renderCache.result==null)){let{result:t,options:r}=this.renderDiffWithHighlighter(e,this.highlighter,!n);this.renderCache={diff:e,options:r,highlighted:n,result:t,renderRange:void 0}}(!t||!n)&&this.asyncHighlight(e).then(({result:t,options:n})=>{this.onHighlightSuccess(e,t,n)})}return this.renderCache.result==null?void 0:this.processDiffResult(this.renderCache.diff,t,this.renderCache.result)}async asyncRender(e,t=Ne){let{result:n}=await this.asyncHighlight(e);return this.processDiffResult(e,t,n)}createPreElement(e,t,n,r,i){let{diffIndicators:a,disableBackground:o,disableLineNumbers:s,overflow:c,themeType:l}=this.getOptionsWithDefaults();return $t({type:`diff`,diffIndicators:a,disableBackground:o,disableLineNumbers:s,overflow:c,themeStyles:n,split:e,themeType:r??l,totalLines:t,customProperties:i})}async asyncHighlight(e){this.computedLang=e.lang??Ue(e.name);let t=this.highlighter!=null&&Kt(this.options.theme??st),n=this.highlighter!=null&&Gt(this.computedLang);return(this.highlighter==null||!t||!n)&&(this.highlighter=await this.initializeHighlighter()),this.renderDiffWithHighlighter(e,this.highlighter)}renderDiffWithHighlighter(e,t,n=!1){let{options:r}=this.getRenderOptions(e),{collapsedContextThreshold:i}=this.getOptionsWithDefaults();return{result:Ke(e,t,r,{forcePlainText:n,expandedHunks:n?!0:void 0,collapsedContextThreshold:i}),options:r}}onHighlightSuccess(e,t,n){if(this.renderCache==null)return;let r=this.renderCache.diff!==e||!this.renderCache.highlighted||!On(this.renderCache.options,n);this.renderCache={diff:e,options:n,highlighted:!0,result:t,renderRange:void 0},r&&this.onRenderUpdate?.()}onHighlightError(e){console.error(e)}processDiffResult(e,t,{code:n,themeStyles:r,baseThemeType:i}){let{diffStyle:a,disableFileHeader:o,expandUnchanged:s,expansionLineCount:c,collapsedContextThreshold:l,hunkSeparators:u}=this.getOptionsWithDefaults();this.diff=e;let d=a===`unified`,f=[],p=[],m=[],h=[],{additionLines:g,deletionLines:_}=n,v={rowCount:0,hunkSeparators:u,additionsContentAST:f,deletionsContentAST:p,unifiedContentAST:m,unifiedGutterAST:Je(),deletionsGutterAST:Je(),additionsGutterAST:Je(),expansionLineCount:c,hunkData:h,incrementRowCount(e=1){v.rowCount+=e},pushToGutter(e,t){switch(e){case`unified`:v.unifiedGutterAST.children.push(t);break;case`deletions`:v.deletionsGutterAST.children.push(t);break;case`additions`:v.additionsGutterAST.children.push(t);break}}},y=Fn(e),b={size:0,side:void 0,increment(){this.size+=1},flush(){if(a!==`unified`){if(this.size<=0||this.side==null){this.side=void 0,this.size=0;return}this.side===`additions`?(v.pushToGutter(`additions`,G(void 0,`buffer`,this.size)),f?.push(yn(this.size))):(v.pushToGutter(`deletions`,G(void 0,`buffer`,this.size)),p?.push(yn(this.size))),this.size=0,this.side=void 0}}},x=(e,t,n,r,i)=>{v.pushToGutter(e,_e(t,n,r,i))};function S(e){b.flush(),a===`unified`?Nn(`unified`,e,v):(Nn(`deletions`,e,v),Nn(`additions`,e,v))}Ge({diff:e,diffStyle:a,startingLine:t.startingLine,totalLines:t.totalLines,expandedHunks:s?!0:this.expandedHunks,collapsedContextThreshold:l,callback:({hunkIndex:t,hunk:n,collapsedBefore:r,collapsedAfter:i,additionLine:o,deletionLine:s,type:c})=>{let l=s==null?o.splitLineIndex:s.splitLineIndex,d=o==null?s.unifiedLineIndex:o.unifiedLineIndex;a===`split`&&c!==`change`&&b.flush(),r>0&&S({hunkIndex:t,collapsedLines:r,rangeSize:Math.max(n?.collapsedBefore??0,0),hunkSpecs:n?.hunkSpecs,isFirstHunk:t===0,isLastHunk:!1,isExpandable:!e.isPartial});let f=a===`unified`?d:l,p={type:c,hunkIndex:t,lineIndex:f,unifiedLineIndex:d,splitLineIndex:l,deletionLine:s,additionLine:o};if(a===`unified`){let n=s==null?void 0:_[s.lineIndex],r=o==null?void 0:g[o.lineIndex];if(n==null&&r==null){let t=`DiffHunksRenderer.processDiffResult: deletionLine and additionLine are null, something is wrong`;throw console.error(t,{file:e.name}),Error(t)}let i=c===`change`?o==null?`change-deletion`:`change-addition`:c,a=this.getUnifiedLineDecoration({type:c,lineType:i,additionLineIndex:o?.lineIndex,deletionLineIndex:s?.lineIndex});x(`unified`,a.gutterLineType,o==null?s.lineNumber:o.lineNumber,`${d},${l}`,a.gutterProperties),r==null?n!=null&&(n=Pn(n,a.contentProperties)):r=Pn(r,a.contentProperties),Mn({diffStyle:`unified`,type:c,deletionLine:n,additionLine:r,unifiedSpan:this.getAnnotations(`unified`,s?.lineNumber,o?.lineNumber,t,f),createAnnotationElement:e=>this.createAnnotationElement(e),context:v});let u=this.getUnifiedInlineRowsForLine?.(p);u!=null&&An(u,v)}else{let n=s==null?void 0:_[s.lineIndex],r=o==null?void 0:g[o.lineIndex],i=this.getSplitLineDecoration({side:`deletions`,type:c,lineIndex:s?.lineIndex}),a=this.getSplitLineDecoration({side:`additions`,type:c,lineIndex:o?.lineIndex});if(n==null&&r==null){let t=`DiffHunksRenderer.processDiffResult: deletionLine and additionLine are null, something is wrong`;throw console.error(t,{file:e.name}),Error(t)}let u=(()=>{if(c===`change`){if(r==null)return`additions`;if(n==null)return`deletions`}})();if(u!=null){if(b.side!=null&&b.side!==u)throw Error(`DiffHunksRenderer.processDiffResult: iterateOverDiff, invalid pending splits`);b.side=u,b.increment()}let d=this.getAnnotations(`split`,s?.lineNumber,o?.lineNumber,t,f);if(d!=null&&b.size>0&&b.flush(),s!=null){let e=Pn(n,i.contentProperties);x(`deletions`,i.gutterLineType,s.lineNumber,`${s.unifiedLineIndex},${l}`,i.gutterProperties),e!=null&&(n=e)}if(o!=null){let e=Pn(r,a.contentProperties);x(`additions`,a.gutterLineType,o.lineNumber,`${o.unifiedLineIndex},${l}`,a.gutterProperties),e!=null&&(r=e)}Mn({diffStyle:`split`,type:c,additionLine:r,deletionLine:n,...d,createAnnotationElement:e=>this.createAnnotationElement(e),context:v});let m=this.getSplitInlineRowsForLine?.(p);m!=null&&jn(m,v,b)}let m=s?.noEOFCR??!1,h=o?.noEOFCR??!1;if(h||m){if(m){let e=c===`context`||c===`context-expanded`?c:`change-deletion`;a===`unified`?(v.unifiedContentAST.push(bn(e)),v.pushToGutter(`unified`,G(e,`metadata`,1))):(v.deletionsContentAST.push(bn(e)),v.pushToGutter(`deletions`,G(e,`metadata`,1)),h||(v.pushToGutter(`additions`,G(void 0,`buffer`,1)),v.additionsContentAST.push(yn(1))))}if(h){let e=c===`context`||c===`context-expanded`?c:`change-addition`;a===`unified`?(v.unifiedContentAST.push(bn(e)),v.pushToGutter(`unified`,G(e,`metadata`,1))):(v.additionsContentAST.push(bn(e)),v.pushToGutter(`additions`,G(e,`metadata`,1)),m||(v.pushToGutter(`deletions`,G(void 0,`buffer`,1)),v.deletionsContentAST.push(yn(1))))}v.incrementRowCount(1)}i>0&&u!==`simple`&&S({hunkIndex:c===`context-expanded`?t:t+1,collapsedLines:i,rangeSize:y,hunkSpecs:void 0,isFirstHunk:!1,isLastHunk:!0,isExpandable:!e.isPartial}),v.incrementRowCount(1)}}),a===`split`&&b.flush();let C=Math.max(wn(e.hunks),e.additionLines.length??0,e.deletionLines.length??0),w=t.bufferBefore>0||t.bufferAfter>0,T=!d&&e.type!==`deleted`,E=!d&&e.type!==`new`,D=v.rowCount>0||w;f=T&&D?f:void 0,p=E&&D?p:void 0,m=d&&D?m:void 0;let O=this.createPreElement(p!=null&&f!=null,C,r,i);return{unifiedGutterAST:d&&D?v.unifiedGutterAST.children:void 0,unifiedContentAST:m,deletionsGutterAST:E&&D?v.deletionsGutterAST.children:void 0,deletionsContentAST:p,additionsGutterAST:T&&D?v.additionsGutterAST.children:void 0,additionsContentAST:f,hunkData:h,preNode:O,themeStyles:r,baseThemeType:i,headerElement:o?void 0:this.renderHeader(this.diff,r,i),totalLines:C,rowCount:v.rowCount,bufferBefore:t.bufferBefore,bufferAfter:t.bufferAfter,css:``}}renderCodeAST(e,t){let n=e===`unified`?t.unifiedGutterAST:e===`deletions`?t.deletionsGutterAST:t.additionsGutterAST,r=e===`unified`?t.unifiedContentAST:e===`deletions`?t.deletionsContentAST:t.additionsContentAST;if(n==null||r==null)return;let i=Je(n);return i.properties.style=`grid-row: span ${t.rowCount}`,[i,rn(r,t.rowCount)]}renderFullAST(e,t=[]){let n=this.getOptionsWithDefaults().hunkSeparators===`line-info`,r=this.renderCodeAST(`unified`,e);if(r!=null)return t.push(W({tagName:`code`,children:r,properties:{"data-code":``,"data-container-size":n?``:void 0,"data-unified":``}})),{...e.preNode,children:t};let i=this.renderCodeAST(`deletions`,e);i!=null&&t.push(W({tagName:`code`,children:i,properties:{"data-code":``,"data-container-size":n?``:void 0,"data-deletions":``}}));let a=this.renderCodeAST(`additions`,e);return a!=null&&t.push(W({tagName:`code`,children:a,properties:{"data-code":``,"data-container-size":n?``:void 0,"data-additions":``}})),{...e.preNode,children:t}}renderFullHTML(e,t=[]){return Ae(this.renderFullAST(e,t))}renderPartialHTML(e,t){return Ae(t==null?e:W({tagName:`code`,children:e,properties:{"data-code":``,"data-container-size":this.getOptionsWithDefaults().hunkSeparators===`line-info`?``:void 0,[`data-${t}`]:``}}))}getAnnotations(e,t,n,r,i){let a={type:`annotation`,hunkIndex:r,lineIndex:i,annotations:[]};if(t!=null)for(let e of this.deletionAnnotations[t]??[])a.annotations.push(nn(e));let o={type:`annotation`,hunkIndex:r,lineIndex:i,annotations:[]};if(n!=null)for(let t of this.additionAnnotations[n]??[])(e===`unified`?a:o).annotations.push(nn(t));if(e===`unified`)return a.annotations.length>0?a:void 0;if(!(o.annotations.length===0&&a.annotations.length===0))return{deletionSpan:a,additionSpan:o}}renderHeader(e,t,n){let{themeType:r}=this.getOptionsWithDefaults();return Xt({fileOrDiff:e,themeStyles:t,themeType:n??r})}};function On(e,t){return at(e.theme,t.theme)&&e.tokenizeMaxLineLength===t.tokenizeMaxLineLength&&e.lineDiffType===t.lineDiffType}function kn(e){return`${e} unmodified line${e>1?`s`:``}`}function An(e,t){for(let n of e)t.unifiedContentAST.push(n.content),t.pushToGutter(`unified`,n.gutter),t.incrementRowCount(1)}function jn(e,t,n){for(let{deletion:r,addition:i}of e){if(r==null&&i==null)continue;let e=r!=null&&i!=null?void 0:r==null?`deletions`:`additions`;(e==null||n.side!==e)&&n.flush(),r!=null&&(t.deletionsContentAST.push(r.content),t.pushToGutter(`deletions`,r.gutter)),i!=null&&(t.additionsContentAST.push(i.content),t.pushToGutter(`additions`,i.gutter)),e!=null&&(n.side=e,n.increment()),t.incrementRowCount(1)}}function Mn({diffStyle:e,type:t,deletionLine:n,additionLine:r,unifiedSpan:i,deletionSpan:a,additionSpan:o,createAnnotationElement:s,context:c}){let l=!1;if(e===`unified`){if(r==null?n!=null&&c.unifiedContentAST.push(n):c.unifiedContentAST.push(r),i!=null){let e=t===`change`?n==null?`change-addition`:`change-deletion`:t;c.unifiedContentAST.push(s(i)),c.pushToGutter(`unified`,G(e,`annotation`,1)),l=!0}}else if(e===`split`){if(n!=null&&c.deletionsContentAST.push(n),r!=null&&c.additionsContentAST.push(r),a!=null){let e=t===`change`?n==null?`context`:`change-deletion`:t;c.deletionsContentAST.push(s(a)),c.pushToGutter(`deletions`,G(e,`annotation`,1)),l=!0}if(o!=null){let e=t===`change`?r==null?`context`:`change-addition`:t;c.additionsContentAST.push(s(o)),c.pushToGutter(`additions`,G(e,`annotation`,1)),l=!0}}l&&c.incrementRowCount(1)}function Nn(e,{hunkIndex:t,collapsedLines:n,rangeSize:r,hunkSpecs:i,isFirstHunk:a,isLastHunk:o,isExpandable:s},c){if(n<=0)return;let l=e===`unified`?c.unifiedContentAST:e===`deletions`?c.deletionsContentAST:c.additionsContentAST;if(c.hunkSeparators===`metadata`){i!=null&&(c.pushToGutter(e,Sn({type:`metadata`,content:i,isFirstHunk:a,isLastHunk:o})),l.push(Sn({type:`metadata`,content:i,isFirstHunk:a,isLastHunk:o})),e!==`additions`&&c.incrementRowCount(1));return}if(c.hunkSeparators===`simple`){t>0&&(c.pushToGutter(e,Sn({type:`simple`,isFirstHunk:a,isLastHunk:!1})),l.push(Sn({type:`simple`,isFirstHunk:a,isLastHunk:!1})),e!==`additions`&&c.incrementRowCount(1));return}let u=Cn(e,t),d=r>c.expansionLineCount,f=s?t:void 0;c.pushToGutter(e,Sn({type:c.hunkSeparators,content:kn(n),expandIndex:f,chunked:d,slotName:u,isFirstHunk:a,isLastHunk:o})),l.push(Sn({type:c.hunkSeparators,content:kn(n),expandIndex:f,chunked:d,slotName:u,isFirstHunk:a,isLastHunk:o})),e!==`additions`&&c.incrementRowCount(1),c.hunkData.push({slotName:u,hunkIndex:t,lines:n,type:e,expandable:s?{up:!a,down:!o,chunked:d}:void 0})}function Pn(e,t){return e==null||e.type!==`element`||t==null?e:{...e,properties:{...e.properties,...t}}}function Fn(e){let t=e.hunks.at(-1);if(t==null||e.isPartial||e.additionLines.length===0||e.deletionLines.length===0)return 0;let n=e.additionLines.length-(t.additionLineIndex+t.additionCount),r=e.deletionLines.length-(t.deletionLineIndex+t.deletionCount);if(n!==r)throw Error(`DiffHunksRenderer.processDiffResult: trailing context mismatch (additions=${n}, deletions=${r}) for ${e.name}`);return Math.min(n,r)}function In(e,t){return e.lineNumber===t.lineNumber&&e.side===t.side&&e.metadata===t.metadata}function Ln(e,t){return e.slotName===t.slotName&&e.hunkIndex===t.hunkIndex&&e.lines===t.lines&&e.type===t.type&&e.expandable?.chunked===t.expandable?.chunked&&e.expandable?.up===t.expandable?.up&&e.expandable?.down===t.expandable?.down}function Rn(e){let t=e[0];if(t!==`+`&&t!==`-`&&t!==` `&&t!==`\\`){console.error(`parseLineType: Invalid firstChar: "${t}", full line: "${e}"`);return}let n=e.substring(1);return{line:n===``?`
`:n,type:t===` `?`context`:t===`\\`?`metadata`:t===`+`?`addition`:`deletion`}}function zn(e,t,n=!1){let r=De.test(e),i=e.split(r?De:Me),a,o=[];for(let e of i){if(r&&!De.test(e)){if(a==null)a=e;else if(n)throw Error(`parsePatchContent: unknown file blob`);else console.error(`parsePatchContent: unknown file blob:`,e);continue}else if(!r&&!Me.test(e)){if(a==null)a=e;else if(n)throw Error(`parsePatchContent: unknown file blob`);else console.error(`parsePatchContent: unknown file blob:`,e);continue}let i=Bn(e,{cacheKey:t==null?void 0:`${t}-${o.length}`,isGitDiff:r,throwOnError:n});i!=null&&o.push(i)}return{patchMetadata:a,files:o}}function Bn(e,{cacheKey:t,isGitDiff:n=De.test(e),oldFile:r,newFile:i,throwOnError:a=!1}={}){let o=0,s=e.split(Ce),c,l=r==null||i==null,u=0,d=0;for(let e of s){let s=e.split(Be),f=s.shift();if(f==null){if(a)throw Error(`parsePatchContent: invalid hunk`);console.error(`parsePatchContent: invalid hunk`,e);continue}let p=f.match(ze),m=0,h=0;if(p==null||c==null){if(c!=null){if(a)throw Error(`parsePatchContent: Invalid hunk`);console.error(`parsePatchContent: Invalid hunk`,e);continue}c={name:``,type:`change`,hunks:[],splitLineCount:0,unifiedLineCount:0,isPartial:l,additionLines:!l&&r!=null&&i!=null?i.contents.split(Be):[],deletionLines:!l&&r!=null&&i!=null?r.contents.split(Be):[],cacheKey:t},c.additionLines.length===1&&i?.contents===``&&(c.additionLines.length=0),c.deletionLines.length===1&&r?.contents===``&&(c.deletionLines.length=0),s.unshift(f);for(let e of s){let t=e.match(n?Le:Fe);if(e.startsWith(`diff --git`)){let[,,t,,n]=e.trim().match(Se)??[];c.name=n.trim(),t!==n&&(c.prevName=t.trim())}else if(t!=null){let[,e,n]=t;e===`---`&&n!==`/dev/null`?(c.prevName=n.trim(),c.name=n.trim()):e===`+++`&&n!==`/dev/null`&&(c.name=n.trim())}else if(n){if(e.startsWith(`new mode `)&&(c.mode=e.replace(`new mode`,``).trim()),e.startsWith(`old mode `)&&(c.prevMode=e.replace(`old mode`,``).trim()),e.startsWith(`new file mode`)&&(c.type=`new`,c.mode=e.replace(`new file mode`,``).trim()),e.startsWith(`deleted file mode`)&&(c.type=`deleted`,c.mode=e.replace(`deleted file mode`,``).trim()),e.startsWith(`similarity index`)&&(e.startsWith(`similarity index 100%`)?c.type=`rename-pure`:c.type=`rename-changed`),e.startsWith(`index `)){let[,t,n,r]=e.trim().match(Re)??[];t!=null&&(c.prevObjectId=t),n!=null&&(c.newObjectId=n),r!=null&&(c.mode=r)}e.startsWith(`rename from `)&&(c.prevName=e.replace(`rename from `,``)),e.startsWith(`rename to `)&&(c.name=e.replace(`rename to `,``).trim())}}continue}let g,_;for(;s.length>0&&(s[s.length-1]===`
`||s[s.length-1]===`\r`||s[s.length-1]===`\r
`||s[s.length-1]===``);)s.pop();let v=parseInt(p[3]),y=parseInt(p[1]);u=l?u:y-1,d=l?d:v-1;let b={collapsedBefore:0,splitLineCount:0,splitLineStart:0,unifiedLineCount:0,unifiedLineStart:0,additionCount:parseInt(p[4]??`1`),additionStart:v,additionLines:m,deletionCount:parseInt(p[2]??`1`),deletionStart:y,deletionLines:h,deletionLineIndex:u,additionLineIndex:d,hunkContent:[],hunkContext:p[5],hunkSpecs:f,noEOFCRAdditions:!1,noEOFCRDeletions:!1};if(isNaN(b.additionCount)||isNaN(b.deletionCount)||isNaN(b.additionStart)||isNaN(b.deletionStart)){if(a)throw Error(`parsePatchContent: invalid hunk metadata`);console.error(`parsePatchContent: invalid hunk metadata`,b);continue}for(let e of s){let t=Rn(e);if(t==null){console.error(`processFile: invalid rawLine:`,e);continue}let{type:n,line:r}=t;if(n===`addition`)(g==null||g.type!==`change`)&&(g=Hn(`change`,u,d),b.hunkContent.push(g)),d++,l&&c.additionLines.push(r),g.additions++,m++,_=`addition`;else if(n===`deletion`)(g==null||g.type!==`change`)&&(g=Hn(`change`,u,d),b.hunkContent.push(g)),u++,l&&c.deletionLines.push(r),g.deletions++,h++,_=`deletion`;else if(n===`context`)(g==null||g.type!==`context`)&&(g=Hn(`context`,u,d),b.hunkContent.push(g)),d++,u++,l&&(c.deletionLines.push(r),c.additionLines.push(r)),g.lines++,_=`context`;else if(n===`metadata`&&g!=null){if(g.type===`context`?(b.noEOFCRAdditions=!0,b.noEOFCRDeletions=!0):_===`deletion`?b.noEOFCRDeletions=!0:_===`addition`&&(b.noEOFCRAdditions=!0),l&&(_===`addition`||_===`context`)){let e=c.additionLines.length-1;e>=0&&(c.additionLines[e]=ot(c.additionLines[e]))}if(l&&(_===`deletion`||_===`context`)){let e=c.deletionLines.length-1;e>=0&&(c.deletionLines[e]=ot(c.deletionLines[e]))}}}b.additionLines=m,b.deletionLines=h,b.collapsedBefore=Math.max(b.additionStart-1-o,0),c.hunks.push(b),o=b.additionStart+b.additionCount-1;for(let e of b.hunkContent)e.type===`context`?(b.splitLineCount+=e.lines,b.unifiedLineCount+=e.lines):(b.splitLineCount+=Math.max(e.additions,e.deletions),b.unifiedLineCount+=e.deletions+e.additions);b.splitLineStart=c.splitLineCount+b.collapsedBefore,b.unifiedLineStart=c.unifiedLineCount+b.collapsedBefore,c.splitLineCount+=b.collapsedBefore+b.splitLineCount,c.unifiedLineCount+=b.collapsedBefore+b.unifiedLineCount}if(c!=null){if(c.hunks.length>0&&!l&&c.additionLines.length>0&&c.deletionLines.length>0){let e=c.hunks[c.hunks.length-1],t=e.additionStart+e.additionCount-1,n=c.additionLines.length,r=Math.max(n-t,0);c.splitLineCount+=r,c.unifiedLineCount+=r}return n||(c.prevName!=null&&c.name!==c.prevName?c.hunks.length>0?c.type=`rename-changed`:c.type=`rename-pure`:i!=null&&i.contents===``?c.type=`deleted`:r!=null&&r.contents===``&&(c.type=`new`)),c.type!==`rename-pure`&&c.type!==`rename-changed`&&(c.prevName=void 0),c}}function Vn(e,t,n=!1){let r=[];for(let i of e.split(Te))try{r.push(zn(i,t==null?void 0:`${t}-${r.length}`,n))}catch(e){if(n)throw e;console.error(e)}return r}function Hn(e,t,n){return e===`change`?{type:`change`,additions:0,deletions:0,additionLineIndex:n,deletionLineIndex:t}:{type:`context`,lines:0,additionLineIndex:n,deletionLineIndex:t}}function Un(e,t,n,r=!1){let i=Bn(Ve(e.name,t.name,e.contents,t.contents,e.header,t.header,n),{cacheKey:(()=>{if(e.cacheKey!=null&&t.cacheKey!=null)return`${e.cacheKey}:${t.cacheKey}`})(),oldFile:e,newFile:t,throwOnError:r});if(i==null)throw Error(`parseDiffFrom: FileInvalid diff -- probably need to fix something -- if the files are the same maybe?`);return i}var Wn=-1,Gn=class{static LoadedCustomComponent=!0;__id=`file-diff:${++Wn}`;fileContainer;spriteSVG;pre;codeUnified;codeDeletions;codeAdditions;bufferBefore;bufferAfter;unsafeCSSStyle;gutterUtilityContent;headerElement;headerPrefix;headerMetadata;separatorCache=new Map;errorWrapper;placeHolder;hunksRenderer;resizeManager;scrollSyncManager;interactionManager;annotationCache=new Map;lineAnnotations=[];deletionFile;additionFile;fileDiff;renderRange;appliedPreAttributes;lastRenderedHeaderHTML;lastRowCount;enabled=!0;constructor(e={theme:st},t,n=!1){this.options=e,this.workerManager=t,this.isContainerManaged=n,this.hunksRenderer=this.createHunksRenderer(e),this.resizeManager=new Wt,this.scrollSyncManager=new vn,this.interactionManager=new Pt(`diff`,Ft(e,typeof e.hunkSeparators==`function`||(e.hunkSeparators??`line-info`)===`line-info`||e.hunkSeparators===`line-info-basic`?this.handleExpandHunk:void 0,this.getLineIndex)),this.workerManager?.subscribeToThemeChanges(this),this.enabled=!0}handleHighlightRender=()=>{this.rerender()};getHunksRendererOptions(e){return{...e,hunkSeparators:typeof e.hunkSeparators==`function`?`custom`:e.hunkSeparators}}createHunksRenderer(e){return new Dn(this.getHunksRendererOptions(e),this.handleHighlightRender,this.workerManager)}getLineIndex=(e,t=`additions`)=>{if(this.fileDiff==null)return;let n=this.fileDiff.hunks.at(-1),r,i;hunkIterator:for(let a of this.fileDiff.hunks){let o=t===`deletions`?a.deletionStart:a.additionStart,s=t===`deletions`?a.deletionCount:a.additionCount,c=a.splitLineStart,l=a.unifiedLineStart;if(e<o){let t=o-e;r=Math.max(l-t,0),i=Math.max(c-t,0);break hunkIterator}if(e>=o+s){if(a===n){let t=e-(o+s);r=l+a.unifiedLineCount+t,i=c+a.splitLineCount+t;break hunkIterator}continue}for(let n of a.hunkContent)if(n.type===`context`)if(e<o+n.lines){let t=e-o;i=c+t,r=l+t;break hunkIterator}else o+=n.lines,c+=n.lines,l+=n.lines;else{let a=t===`deletions`?n.deletions:n.additions;if(e<o+a){let a=e-o;r=l+(t===`additions`?n.deletions:0)+a,i=c+a;break hunkIterator}else o+=a,c+=Math.max(n.deletions,n.additions),l+=n.deletions+n.additions}break hunkIterator}if(!(r==null||i==null))return[r,i]};setOptions(e){e!=null&&(this.options=e,this.hunksRenderer.setOptions(this.getHunksRendererOptions(e)),this.interactionManager.setOptions(Ft(e,typeof e.hunkSeparators==`function`||(e.hunkSeparators??`line-info`)===`line-info`||e.hunkSeparators===`line-info-basic`?this.handleExpandHunk:void 0,this.getLineIndex)))}mergeOptions(e){this.options={...this.options,...e}}setThemeType(e){if((this.options.themeType??`system`)!==e&&(this.mergeOptions({themeType:e}),this.hunksRenderer.setThemeType(e),this.headerElement!=null&&(e===`system`?delete this.headerElement.dataset.themeType:this.headerElement.dataset.themeType=e),this.pre!=null))switch(e){case`system`:delete this.pre.dataset.themeType;break;case`light`:case`dark`:this.pre.dataset.themeType=e;break}}getHoveredLine=()=>this.interactionManager.getHoveredLine();setLineAnnotations(e){this.lineAnnotations=e}canPartiallyRender(e,t,n){return!(e||t||n||typeof this.options.hunkSeparators==`function`)}setSelectedLines(e){this.interactionManager.setSelection(e)}cleanUp(e=!1){this.resizeManager.cleanUp(),this.interactionManager.cleanUp(),this.scrollSyncManager.cleanUp(),this.workerManager?.unsubscribeToThemeChanges(this),this.renderRange=void 0,this.isContainerManaged||this.fileContainer?.remove(),this.fileContainer?.shadowRoot!=null&&(this.fileContainer.shadowRoot.innerHTML=``),this.fileContainer=void 0,this.pre!=null&&(this.pre.innerHTML=``,this.pre=void 0),this.codeUnified=void 0,this.codeDeletions=void 0,this.codeAdditions=void 0,this.bufferBefore=void 0,this.bufferAfter=void 0,this.appliedPreAttributes=void 0,this.headerElement=void 0,this.headerPrefix=void 0,this.headerMetadata=void 0,this.lastRenderedHeaderHTML=void 0,this.errorWrapper=void 0,this.spriteSVG=void 0,this.lastRowCount=void 0,e?this.hunksRenderer.recycle():(this.hunksRenderer.cleanUp(),this.workerManager=void 0,this.fileDiff=void 0,this.deletionFile=void 0,this.additionFile=void 0),this.enabled=!1}virtualizedSetup(){this.enabled=!0,this.workerManager?.subscribeToThemeChanges(this)}hydrate(e){let{overflow:t=`scroll`,diffStyle:n=`split`}=this.options,{fileContainer:r,prerenderedHTML:i}=e;gn(r,i);for(let e of r.shadowRoot?.children??[]){if(e instanceof SVGElement){this.spriteSVG=e;continue}if(e instanceof HTMLElement){if(e instanceof HTMLPreElement){this.pre=e;for(let t of e.children)!(t instanceof HTMLElement)||t.tagName.toLowerCase()!==`code`||(`deletions`in t.dataset&&(this.codeDeletions=t),`additions`in t.dataset&&(this.codeAdditions=t),`unified`in t.dataset&&(this.codeUnified=t));continue}if(`diffsHeader`in e.dataset){this.headerElement=e;continue}if(e instanceof HTMLStyleElement&&e.hasAttribute(`data-unsafe-css`)){this.unsafeCSSStyle=e;continue}}}if(this.pre!=null&&this.syncCodeNodesFromPre(this.pre),this.pre==null)this.render(e);else{let{lineAnnotations:i,oldFile:a,newFile:o,fileDiff:s}=e;this.fileContainer=r,delete this.pre.dataset.dehydrated,this.lineAnnotations=i??this.lineAnnotations,this.additionFile=o,this.deletionFile=a,this.fileDiff=s??(a!=null&&o!=null?Un(a,o):void 0),this.hunksRenderer.hydrate(this.fileDiff),this.renderAnnotations(),this.renderGutterUtility(),this.injectUnsafeCSS(),this.interactionManager.setup(this.pre),this.resizeManager.setup(this.pre,t===`wrap`),t===`scroll`&&n===`split`&&this.scrollSyncManager.setup(this.pre,this.codeDeletions,this.codeAdditions)}}rerender(){!this.enabled||this.fileDiff==null&&this.additionFile==null&&this.deletionFile==null||this.render({forceRender:!0,renderRange:this.renderRange})}handleExpandHunk=(e,t,n)=>{this.expandHunk(e,t,n)};expandHunk=(e,t,n)=>{this.hunksRenderer.expandHunk(e,t,n),this.rerender()};render({oldFile:e,newFile:t,fileDiff:n,forceRender:r=!1,lineAnnotations:i,fileContainer:a,containerWrapper:o,renderRange:s}){if(!this.enabled)throw Error(`FileDiff.render: attempting to call render after cleaned up`);let{collapsed:c=!1}=this.options,l=c?void 0:s,u=e!=null&&t!=null&&(!rt(e,this.deletionFile)||!rt(t,this.additionFile)),d=n!=null&&n!==this.fileDiff,f=i!=null&&(i.length>0||this.lineAnnotations.length>0)?i!==this.lineAnnotations:!1;if(!c&&qt(l,this.renderRange)&&!r&&!f&&(n!=null&&n===this.fileDiff||n==null&&!u))return!1;let{renderRange:p}=this;if(this.renderRange=l,this.deletionFile=e,this.additionFile=t,n==null?e!=null&&t!=null&&u&&(d=!0,this.fileDiff=Un(e,t)):this.fileDiff=n,i!=null&&this.setLineAnnotations(i),this.fileDiff==null)return!1;this.hunksRenderer.setOptions({...this.options,hunkSeparators:typeof this.options.hunkSeparators==`function`?`custom`:this.options.hunkSeparators}),this.hunksRenderer.setLineAnnotations(this.lineAnnotations);let{diffStyle:m=`split`,disableErrorHandling:h=!1,disableFileHeader:g=!1,overflow:_=`scroll`}=this.options;if(g&&(this.headerElement!=null&&(this.headerElement.remove(),this.headerElement=void 0,this.lastRenderedHeaderHTML=void 0),this.headerPrefix!=null&&(this.headerPrefix.remove(),this.headerPrefix=void 0),this.headerMetadata!=null&&(this.headerMetadata.remove(),this.headerMetadata=void 0)),a=this.getOrCreateFileContainer(a,o),c){this.removeRenderedCode(),this.clearAuxiliaryNodes();try{let e=this.hunksRenderer.renderDiff(this.fileDiff,we);e?.headerElement!=null&&this.applyHeaderToDOM(e.headerElement,a),this.renderSeparators([]),this.injectUnsafeCSS()}catch(e){if(h)throw e;console.error(e),e instanceof Error&&this.applyErrorToDOM(e,a)}return!0}try{let e=this.getOrCreatePreNode(a);if(!(this.canPartiallyRender(r,f,u||d)&&this.applyPartialRender({previousRenderRange:p,renderRange:l}))){let t=this.hunksRenderer.renderDiff(this.fileDiff,l);if(t==null)return this.workerManager?.isInitialized()===!1&&this.workerManager.initialize().then(()=>this.rerender()),!1;t.headerElement!=null&&this.applyHeaderToDOM(t.headerElement,a),t.additionsContentAST!=null||t.deletionsContentAST!=null||t.unifiedContentAST!=null?this.applyHunksToDOM(e,t):this.pre!=null&&(this.pre.remove(),this.pre=void 0),this.renderSeparators(t.hunkData)}this.applyBuffers(e,l),this.injectUnsafeCSS(),this.renderAnnotations(),this.renderGutterUtility(),this.interactionManager.setup(e),this.resizeManager.setup(e,_===`wrap`),_===`scroll`&&m===`split`?this.scrollSyncManager.setup(e,this.codeDeletions,this.codeAdditions):this.scrollSyncManager.cleanUp()}catch(e){if(h)throw e;console.error(e),e instanceof Error&&this.applyErrorToDOM(e,a)}return!0}removeRenderedCode(){this.resizeManager.cleanUp(),this.scrollSyncManager.cleanUp(),this.interactionManager.cleanUp(),this.bufferBefore?.remove(),this.bufferBefore=void 0,this.bufferAfter?.remove(),this.bufferAfter=void 0,this.codeUnified?.remove(),this.codeUnified=void 0,this.codeDeletions?.remove(),this.codeDeletions=void 0,this.codeAdditions?.remove(),this.codeAdditions=void 0,this.pre?.remove(),this.pre=void 0,this.appliedPreAttributes=void 0,this.lastRowCount=void 0}clearAuxiliaryNodes(){for(let{element:e}of this.separatorCache.values())e.remove();this.separatorCache.clear();for(let{element:e}of this.annotationCache.values())e.remove();this.annotationCache.clear(),this.gutterUtilityContent?.remove(),this.gutterUtilityContent=void 0}renderPlaceholder(e){if(this.fileContainer==null)return!1;if(this.cleanChildNodes(),this.placeHolder==null){let e=this.fileContainer.shadowRoot??this.fileContainer.attachShadow({mode:`open`});this.placeHolder=document.createElement(`div`),this.placeHolder.dataset.placeholder=``,e.appendChild(this.placeHolder)}return this.placeHolder.style.setProperty(`height`,`${e}px`),!0}cleanChildNodes(){this.resizeManager.cleanUp(),this.scrollSyncManager.cleanUp(),this.interactionManager.cleanUp(),this.bufferAfter?.remove(),this.bufferBefore?.remove(),this.codeAdditions?.remove(),this.codeDeletions?.remove(),this.codeUnified?.remove(),this.errorWrapper?.remove(),this.headerElement?.remove(),this.gutterUtilityContent?.remove(),this.headerPrefix?.remove(),this.headerMetadata?.remove(),this.pre?.remove(),this.spriteSVG?.remove(),this.unsafeCSSStyle?.remove(),this.bufferAfter=void 0,this.bufferBefore=void 0,this.codeAdditions=void 0,this.codeDeletions=void 0,this.codeUnified=void 0,this.errorWrapper=void 0,this.headerElement=void 0,this.gutterUtilityContent=void 0,this.headerPrefix=void 0,this.headerMetadata=void 0,this.pre=void 0,this.spriteSVG=void 0,this.unsafeCSSStyle=void 0,this.lastRenderedHeaderHTML=void 0,this.lastRowCount=void 0}renderSeparators(e){let{hunkSeparators:t}=this.options;if(this.isContainerManaged||this.fileContainer==null||typeof t!=`function`){for(let{element:e}of this.separatorCache.values())e.remove();this.separatorCache.clear();return}let n=new Map(this.separatorCache);for(let r of e){let e=r.slotName,i=this.separatorCache.get(e);if(i==null||!Ln(r,i.hunkData)){i?.element.remove();let n=document.createElement(`div`);n.style.display=`contents`,n.slot=r.slotName;let a=t(r,this);a!=null&&n.appendChild(a),this.fileContainer.appendChild(n),i={element:n,hunkData:r},this.separatorCache.set(e,i)}n.delete(e)}for(let[e,{element:t}]of n.entries())this.separatorCache.delete(e),t.remove()}renderAnnotations(){if(this.isContainerManaged||this.fileContainer==null){for(let{element:e}of this.annotationCache.values())e.remove();this.annotationCache.clear();return}let e=new Map(this.annotationCache),{renderAnnotation:t}=this.options;if(t!=null&&this.lineAnnotations.length>0)for(let[n,r]of this.lineAnnotations.entries()){let i=`${n}-${nn(r)}`,a=this.annotationCache.get(i);if(a==null||!In(r,a.annotation)){a?.element.remove();let e=t(r);if(e==null)continue;a={element:ln(nn(r)),annotation:r},a.element.appendChild(e),this.fileContainer.appendChild(a.element),this.annotationCache.set(i,a)}e.delete(i)}for(let[t,{element:n}]of e.entries())this.annotationCache.delete(t),n.remove()}renderGutterUtility(){let e=this.options.renderGutterUtility??this.options.renderHoverUtility;if(this.fileContainer==null||e==null){this.gutterUtilityContent?.remove(),this.gutterUtilityContent=void 0;return}let t=e(this.interactionManager.getHoveredLine);if(t!=null&&this.gutterUtilityContent!=null)return;if(t==null){this.gutterUtilityContent?.remove(),this.gutterUtilityContent=void 0;return}let n=un();n.appendChild(t),this.fileContainer.appendChild(n),this.gutterUtilityContent=n}getOrCreateFileContainer(e,t){let n=this.fileContainer;if(this.fileContainer=e??this.fileContainer??document.createElement(`diffs-container`),n!=null&&n!==this.fileContainer&&(this.lastRenderedHeaderHTML=void 0,this.headerElement=void 0),t!=null&&this.fileContainer.parentNode!==t&&t.appendChild(this.fileContainer),this.spriteSVG==null){let e=document.createElement(`div`);e.innerHTML=an;let t=e.firstChild;t instanceof SVGElement&&(this.spriteSVG=t,this.fileContainer.shadowRoot?.appendChild(this.spriteSVG))}return this.fileContainer}getFileContainer(){return this.fileContainer}getOrCreatePreNode(e){let t=e.shadowRoot??e.attachShadow({mode:`open`});return this.pre==null?(this.pre=document.createElement(`pre`),this.appliedPreAttributes=void 0,this.codeUnified=void 0,this.codeDeletions=void 0,this.codeAdditions=void 0,t.appendChild(this.pre)):this.pre.parentNode!==t&&(t.appendChild(this.pre),this.appliedPreAttributes=void 0),this.placeHolder?.remove(),this.placeHolder=void 0,this.pre}syncCodeNodesFromPre(e){this.codeUnified=void 0,this.codeDeletions=void 0,this.codeAdditions=void 0;for(let t of Array.from(e.children))t instanceof HTMLElement&&(`unified`in t.dataset?this.codeUnified=t:`deletions`in t.dataset?this.codeDeletions=t:`additions`in t.dataset&&(this.codeAdditions=t))}applyHeaderToDOM(e,t){this.cleanupErrorWrapper(),this.placeHolder?.remove(),this.placeHolder=void 0;let n=Ae(e);if(n!==this.lastRenderedHeaderHTML){let e=document.createElement(`div`);e.innerHTML=n;let r=e.firstElementChild;if(!(r instanceof HTMLElement))return;this.headerElement==null?t.shadowRoot?.prepend(r):t.shadowRoot?.replaceChild(r,this.headerElement),this.headerElement=r,this.lastRenderedHeaderHTML=n}if(this.isContainerManaged)return;let{renderHeaderPrefix:r,renderHeaderMetadata:i}=this.options;this.headerPrefix!=null&&this.headerPrefix.remove(),this.headerMetadata!=null&&this.headerMetadata.remove();let a=r?.({deletionFile:this.deletionFile,additionFile:this.additionFile,fileDiff:this.fileDiff})??void 0,o=i?.({deletionFile:this.deletionFile,additionFile:this.additionFile,fileDiff:this.fileDiff})??void 0;a!=null&&(this.headerPrefix=document.createElement(`div`),this.headerPrefix.slot=Ee,a instanceof Element?this.headerPrefix.appendChild(a):this.headerPrefix.innerText=`${a}`,t.appendChild(this.headerPrefix)),o!=null&&(this.headerMetadata=document.createElement(`div`),this.headerMetadata.slot=Qe,o instanceof Element?this.headerMetadata.appendChild(o):this.headerMetadata.innerText=`${o}`,t.appendChild(this.headerMetadata))}injectUnsafeCSS(){if(this.fileContainer?.shadowRoot==null)return;let{unsafeCSS:e}=this.options;e==null||e===``||(this.unsafeCSSStyle??(this.unsafeCSSStyle=dn(),this.fileContainer.shadowRoot.appendChild(this.unsafeCSSStyle)),this.unsafeCSSStyle.innerText=mn(e))}applyHunksToDOM(e,t){let{overflow:n=`scroll`}=this.options,r=(this.options.hunkSeparators??`line-info`)===`line-info`,i=n===`wrap`?t.rowCount:void 0;this.cleanupErrorWrapper(),this.applyPreNodeAttributes(e,t);let a=!1,o=[],s=this.hunksRenderer.renderCodeAST(`unified`,t),c=this.hunksRenderer.renderCodeAST(`deletions`,t),l=this.hunksRenderer.renderCodeAST(`additions`,t);s==null?c!=null||l!=null?(c==null?(this.codeDeletions?.remove(),this.codeDeletions=void 0):(a=this.codeDeletions==null||this.codeUnified!=null,this.codeUnified?.remove(),this.codeUnified=void 0,this.codeDeletions=hn({code:this.codeDeletions,columnType:`deletions`,rowSpan:i,containerSize:r}),this.codeDeletions.innerHTML=this.hunksRenderer.renderPartialHTML(c),o.push(this.codeDeletions)),l==null?(this.codeAdditions?.remove(),this.codeAdditions=void 0):(a=a||this.codeAdditions==null||this.codeUnified!=null,this.codeUnified?.remove(),this.codeUnified=void 0,this.codeAdditions=hn({code:this.codeAdditions,columnType:`additions`,rowSpan:i,containerSize:r}),this.codeAdditions.innerHTML=this.hunksRenderer.renderPartialHTML(l),o.push(this.codeAdditions))):(this.codeUnified?.remove(),this.codeUnified=void 0,this.codeDeletions?.remove(),this.codeDeletions=void 0,this.codeAdditions?.remove(),this.codeAdditions=void 0):(a=this.codeUnified==null||this.codeAdditions!=null||this.codeDeletions!=null,this.codeDeletions?.remove(),this.codeDeletions=void 0,this.codeAdditions?.remove(),this.codeAdditions=void 0,this.codeUnified=hn({code:this.codeUnified,columnType:`unified`,rowSpan:i,containerSize:r}),this.codeUnified.innerHTML=this.hunksRenderer.renderPartialHTML(s),o.push(this.codeUnified)),o.length===0?e.textContent=``:a&&e.replaceChildren(...o),this.lastRowCount=t.rowCount}applyPartialRender({previousRenderRange:e,renderRange:t}){let{pre:n,codeUnified:r,codeAdditions:i,codeDeletions:a,options:{diffStyle:o=`split`}}=this;if(n==null||e==null||t==null||!Number.isFinite(e.totalLines)||!Number.isFinite(t.totalLines)||this.lastRowCount==null)return!1;let s=this.getCodeColumns(o,r,a,i);if(s==null)return!1;let c=e.startingLine,l=t.startingLine,u=c+e.totalLines,d=l+t.totalLines,f=Math.max(c,l),p=Math.min(u,d);if(p<=f)return!1;let m=Math.max(0,f-c),h=Math.max(0,u-p),g=this.trimColumns({columns:s,trimStart:m,trimEnd:h,previousStart:c,overlapStart:f,overlapEnd:p,diffStyle:o});if(g<0)throw Error(`applyPartialRender: failed to trim to overlap`);if(this.lastRowCount<g)throw Error(`applyPartialRender: trimmed beyond DOM row count`);let _=this.lastRowCount-g,v=(e,t)=>{if(!(t<=0||this.fileDiff==null))return this.hunksRenderer.renderDiff(this.fileDiff,{startingLine:e,totalLines:t,bufferBefore:0,bufferAfter:0})},y=v(l,Math.max(f-l,0));if(y==null&&l<f)return!1;let b=v(p,Math.max(d-p,0));if(b==null&&d>p)return!1;let x=(e,t)=>{if(e!=null){if(o===`unified`&&!Array.isArray(s))this.insertPartialHTML(o,s,e,t);else if(o===`split`&&Array.isArray(s))this.insertPartialHTML(o,s,e,t);else throw Error(`FileDiff.applyPartialRender.applyChunk: invalid chunk application`);_+=e.rowCount}};return this.cleanupErrorWrapper(),x(y,`afterbegin`),x(b,`beforeend`),this.lastRowCount!==_&&(this.applyRowSpan(o,s,_),this.lastRowCount=_),!0}insertPartialHTML(e,t,n,r){if(e===`unified`&&!Array.isArray(t)){let e=this.hunksRenderer.renderCodeAST(`unified`,n);this.renderPartialColumn(t,e,r)}else if(e===`split`&&Array.isArray(t)){let e=this.hunksRenderer.renderCodeAST(`deletions`,n),i=this.hunksRenderer.renderCodeAST(`additions`,n);this.renderPartialColumn(t[0],e,r),this.renderPartialColumn(t[1],i,r)}else throw Error(`FileDiff.insertPartialHTML: Invalid argument composition`)}renderPartialColumn(e,t,n){if(e==null||t==null)return;let r=Kn(t[0]),i=Kn(t[1]);if(r==null||i==null)throw Error(`FileDiff.insertPartialHTML: Unexpected AST structure`);let a=i.at(0);n===`beforeend`&&a?.type===`element`&&typeof a.properties[`data-buffer-size`]==`number`&&this.mergeBuffersIfNecessary(a.properties[`data-buffer-size`],e.content.children[e.content.children.length-1],e.gutter.children[e.gutter.children.length-1],r,i,!0);let o=i.at(-1);n===`afterbegin`&&o?.type===`element`&&typeof o.properties[`data-buffer-size`]==`number`&&this.mergeBuffersIfNecessary(o.properties[`data-buffer-size`],e.content.children[0],e.gutter.children[0],r,i,!1),e.gutter.insertAdjacentHTML(n,this.hunksRenderer.renderPartialHTML(r)),e.content.insertAdjacentHTML(n,this.hunksRenderer.renderPartialHTML(i))}mergeBuffersIfNecessary(e,t,n,r,i,a){if(!(t instanceof HTMLElement)||!(n instanceof HTMLElement))return;let o=this.getBufferSize(t.dataset);o!=null&&(a?(r.shift(),i.shift()):(r.pop(),i.pop()),this.updateBufferSize(t,o+e),this.updateBufferSize(n,o+e))}applyRowSpan(e,t,n){let r=e=>{e!=null&&(e.gutter.style.setProperty(`grid-row`,`span ${n}`),e.content.style.setProperty(`grid-row`,`span ${n}`))};if(e===`unified`&&!Array.isArray(t))r(t);else if(e===`split`&&Array.isArray(t))r(t[0]),r(t[1]);else throw Error(`dun fuuuuked up`)}trimColumnRows(e,t,n){let r=0,i=0,a=0,o=!1,s=n>=0;if(e==null)return 0;let c=Array.from(e.content.children),l=Array.from(e.gutter.children);if(c.length!==l.length)throw Error(`FileDiff.trimColumnRows: columns do not match`);for(;a<c.length&&!(t<=0&&!s&&!o);){let e=l[a],u=c[a];if(a++,!(e instanceof HTMLElement)||!(u instanceof HTMLElement))throw console.error({gutterElement:e,contentElement:u}),Error(`FileDiff.trimColumnRows: invalid row elements`);if(o&&(o=!1,e.dataset.gutterBuffer===`annotation`&&`lineAnnotation`in u.dataset||e.dataset.gutterBuffer===`metadata`&&`noNewline`in u.dataset)){e.remove(),u.remove(),i++;continue}if(`lineIndex`in e.dataset&&`lineIndex`in u.dataset){(t>0||s&&r>=n)&&(e.remove(),u.remove(),t>0&&(t--,t===0&&(o=!0)),i++),r++;continue}if(`separator`in e.dataset&&`separator`in u.dataset){(t>0||s&&r>=n)&&(e.remove(),u.remove(),i++);continue}if(e.dataset.gutterBuffer===`annotation`&&`lineAnnotation`in u.dataset){(t>0||s&&r>=n)&&(e.remove(),u.remove(),i++);continue}if(e.dataset.gutterBuffer===`metadata`&&`noNewline`in u.dataset){(t>0||s&&r>=n)&&(e.remove(),u.remove(),i++);continue}if(e.dataset.gutterBuffer===`buffer`&&`contentBuffer`in u.dataset){let a=this.getBufferSize(u.dataset);if(a==null)throw Error(`FileDiff.trimColumnRows: invalid element`);if(t>0){let n=Math.min(t,a),r=a-n;r>0?(this.updateBufferSize(e,r),this.updateBufferSize(u,r),i+=n):(e.remove(),u.remove(),i+=a),t-=n}else if(s){let t=r,o=r+a-1;if(n<=t)e.remove(),u.remove(),i+=a;else if(n<=o){let t=o-n+1,r=a-t;this.updateBufferSize(e,r),this.updateBufferSize(u,r),i+=t}}r+=a;continue}throw console.error({gutterElement:e,contentElement:u}),Error(`FileDiff.trimColumnRows: unknown row elements`)}return i}trimColumns({columns:e,diffStyle:t,overlapEnd:n,overlapStart:r,previousStart:i,trimEnd:a,trimStart:o}){let s=Math.max(0,r-i),c=n-i;if(c<0)throw Error(`FileDiff.trimColumns: overlap ends before previous`);let l=o>0,u=a>0;if(!l&&!u)return 0;let d=l?s:0,f=u?c:-1;if(t===`unified`&&!Array.isArray(e))return this.trimColumnRows(e,d,f);if(t===`split`&&Array.isArray(e)){let t=this.trimColumnRows(e[0],d,f),n=this.trimColumnRows(e[1],d,f);if(e[0]!=null&&e[1]!=null&&t!==n)throw Error(`FileDiff.trimColumns: split columns out of sync`);return e[0]==null?n:t}else throw console.error({diffStyle:t,columns:e}),Error(`FileDiff.trimColumns: Invalid columns for diffType`)}getBufferSize(e){let t=Number.parseInt(e?.bufferSize??``,10);return Number.isNaN(t)?void 0:t}updateBufferSize(e,t){e.dataset.bufferSize=`${t}`,e.style.setProperty(`grid-row`,`span ${t}`),e.style.setProperty(`min-height`,`calc(${t} * 1lh)`)}getCodeColumns(e,t,n,r){function i(e){if(e==null)return;let t=e.children[0],n=e.children[1];if(!(!(t instanceof HTMLElement)||!(n instanceof HTMLElement)||t.dataset.gutter==null||n.dataset.content==null))return{gutter:t,content:n}}if(e===`unified`)return i(t);{let e=i(n),t=i(r);return e!=null||t!=null?[e,t]:void 0}}applyBuffers(e,t){let{disableVirtualizationBuffers:n=!1}=this.options;if(n||t==null){this.bufferBefore!=null&&(this.bufferBefore.remove(),this.bufferBefore=void 0),this.bufferAfter!=null&&(this.bufferAfter.remove(),this.bufferAfter=void 0);return}t.bufferBefore>0?(this.bufferBefore??(this.bufferBefore=document.createElement(`div`),this.bufferBefore.dataset.virtualizerBuffer=`before`,e.before(this.bufferBefore)),this.bufferBefore.style.setProperty(`height`,`${t.bufferBefore}px`),this.bufferBefore.style.setProperty(`contain`,`strict`)):this.bufferBefore!=null&&(this.bufferBefore.remove(),this.bufferBefore=void 0),t.bufferAfter>0?(this.bufferAfter??(this.bufferAfter=document.createElement(`div`),this.bufferAfter.dataset.virtualizerBuffer=`after`,e.after(this.bufferAfter)),this.bufferAfter.style.setProperty(`height`,`${t.bufferAfter}px`),this.bufferAfter.style.setProperty(`contain`,`strict`)):this.bufferAfter!=null&&(this.bufferAfter.remove(),this.bufferAfter=void 0)}applyPreNodeAttributes(e,{themeStyles:t,baseThemeType:n,additionsContentAST:r,deletionsContentAST:i,totalLines:a},o){let{diffIndicators:s=`bars`,disableBackground:c=!1,disableLineNumbers:l=!1,overflow:u=`scroll`,themeType:d=`system`,diffStyle:f=`split`}=this.options,p={type:`diff`,diffIndicators:s,disableBackground:c,disableLineNumbers:l,overflow:u,split:f===`unified`?!1:r!=null&&i!=null,themeStyles:t,themeType:n??d,totalLines:a,customProperties:o};on(p,this.appliedPreAttributes)||(_n(e,p),this.appliedPreAttributes=p)}applyErrorToDOM(e,t){this.cleanupErrorWrapper();let n=this.getOrCreatePreNode(t);n.innerHTML=``,n.remove(),this.pre=void 0,this.appliedPreAttributes=void 0;let r=t.shadowRoot??t.attachShadow({mode:`open`});this.errorWrapper??=document.createElement(`div`),this.errorWrapper.dataset.errorWrapper=``,this.errorWrapper.innerHTML=``,r.appendChild(this.errorWrapper);let i=document.createElement(`div`);i.dataset.errorMessage=``,i.innerText=e.message,this.errorWrapper.appendChild(i);let a=document.createElement(`pre`);a.dataset.errorStack=``,a.innerText=e.stack??`No Error Stack`,this.errorWrapper.appendChild(a)}cleanupErrorWrapper(){this.errorWrapper?.remove(),this.errorWrapper=void 0}};function Kn(e){if(!(e==null||e.type!==`element`))return e.children??[]}function qn({side:e,lineNumber:t,conflictIndex:n}){return`merge-conflict-action-${e}-${t}-${n}`}function Jn(e){if(e.incomingLineNumber!=null)return{side:`additions`,lineNumber:e.incomingLineNumber};if(e.currentLineNumber!=null)return{side:`deletions`,lineNumber:e.currentLineNumber}}function Yn(e,t){let n={...ve,...t};return n.hunkSeparatorHeight=Xn(e,t?.hunkSeparatorHeight),n}function Xn(e,t){if(t!=null)return t;switch(e){case`simple`:return 4;case`metadata`:case`line-info`:case`line-info-basic`:case`custom`:return 32}}var Zn=-1,Qn=class extends Gn{__id=`little-virtualized-file-diff:${++Zn}`;top;height=0;metrics;heightCache=new Map;isVisible=!1;virtualizer;constructor(e,t,n,r,i=!1){super(e,r,i);let{hunkSeparators:a=`line-info`}=this.options;this.virtualizer=t,this.metrics=Yn(typeof a==`function`?`custom`:a,n)}getLineHeight(e,t=!1){let n=this.heightCache.get(e);if(n!=null)return n;let r=t?2:1;return this.metrics.lineHeight*r}setOptions(e){if(e==null)return;let t=this.options.diffStyle,n=this.options.overflow,r=this.options.collapsed;super.setOptions(e),(t!==this.options.diffStyle||n!==this.options.overflow||r!==this.options.collapsed)&&(this.heightCache.clear(),this.computeApproximateSize(),this.renderRange=void 0),this.virtualizer.instanceChanged(this)}reconcileHeights(){let{overflow:e=`scroll`}=this.options;if(this.fileContainer!=null&&(this.top=this.virtualizer.getOffsetInScrollContainer(this.fileContainer)),this.fileContainer==null||this.fileDiff==null){this.height=0;return}if(e===`scroll`&&this.lineAnnotations.length===0&&!this.virtualizer.config.resizeDebugging)return;let t=this.getDiffStyle(),n=!1,r=t===`split`?[this.codeDeletions,this.codeAdditions]:[this.codeUnified];for(let e of r){if(e==null)continue;let r=e.children[1];if(r instanceof HTMLElement)for(let e of r.children){if(!(e instanceof HTMLElement))continue;let r=e.dataset.lineIndex;if(r==null)continue;let i=er(r,t),a=e.getBoundingClientRect().height,o=!1;e.nextElementSibling instanceof HTMLElement&&(`lineAnnotation`in e.nextElementSibling.dataset||`noNewline`in e.nextElementSibling.dataset)&&(`noNewline`in e.nextElementSibling.dataset&&(o=!0),a+=e.nextElementSibling.getBoundingClientRect().height);let s=this.getLineHeight(i,o);a!==s&&(n=!0,a===this.metrics.lineHeight*(o?2:1)?this.heightCache.delete(i):this.heightCache.set(i,a))}}(n||this.virtualizer.config.resizeDebugging)&&this.computeApproximateSize()}onRender=e=>this.fileContainer==null?!1:(e&&(this.top=this.virtualizer.getOffsetInScrollContainer(this.fileContainer)),this.render());cleanUp(){this.fileContainer!=null&&this.virtualizer.disconnect(this.fileContainer),super.cleanUp()}expandHunk=(e,t,n)=>{this.hunksRenderer.expandHunk(e,t,n),this.computeApproximateSize(),this.renderRange=void 0,this.virtualizer.instanceChanged(this)};setVisibility(e){this.fileContainer!=null&&(this.renderRange=void 0,e&&!this.isVisible?(this.top=this.virtualizer.getOffsetInScrollContainer(this.fileContainer),this.isVisible=!0):!e&&this.isVisible&&(this.isVisible=!1,this.rerender()))}computeApproximateSize(){let e=this.height===0;if(this.height=0,this.fileDiff==null)return;let{disableFileHeader:t=!1,expandUnchanged:n=!1,collapsed:r=!1,collapsedContextThreshold:i=1,hunkSeparators:a=`line-info`}=this.options,{diffHeaderHeight:o,fileGap:s,hunkSeparatorHeight:c}=this.metrics,l=this.getDiffStyle(),u=a!==`simple`&&a!==`metadata`&&a!==`line-info-basic`?s:0;if(t?a!==`simple`&&a!==`metadata`&&(this.height+=s):this.height+=o,!r&&(Ge({diff:this.fileDiff,diffStyle:l,expandedHunks:n?!0:this.hunksRenderer.getExpandedHunksMap(),collapsedContextThreshold:i,callback:({hunkIndex:e,collapsedBefore:t,collapsedAfter:n,deletionLine:r,additionLine:i})=>{let o=i==null?r.splitLineIndex:i.splitLineIndex,s=i==null?r.unifiedLineIndex:i.unifiedLineIndex,d=(i?.noEOFCR??!1)||(r?.noEOFCR??!1);t>0&&(e>0&&(this.height+=u),this.height+=c+u),this.height+=this.getLineHeight(l===`split`?o:s,d),n>0&&a!==`simple`&&(this.height+=u+c)}}),this.fileDiff.hunks.length>0&&(this.height+=s),this.fileContainer!=null&&this.virtualizer.config.resizeDebugging&&!e)){let e=this.fileContainer.getBoundingClientRect();e.height===this.height?console.log(`VirtualizedFileDiff.computeApproximateSize: computed height IS CORRECT`):console.log(`VirtualizedFileDiff.computeApproximateSize: computed height doesnt match`,{name:this.fileDiff.name,elementHeight:e.height,computedHeight:this.height})}}render({fileContainer:e,oldFile:t,newFile:n,fileDiff:r,...i}={}){let a=this.fileContainer==null;if(this.fileDiff??=r??(t!=null&&n!=null?Un(t,n):void 0),e=this.getOrCreateFileContainer(e),this.fileDiff==null)return console.error(`VirtualizedFileDiff.render: attempting to virtually render when we dont have the correct data`),!1;if(a?(this.computeApproximateSize(),this.virtualizer.connect(e,this),this.top??=this.virtualizer.getOffsetInScrollContainer(e),this.isVisible=this.virtualizer.isInstanceVisible(this.top,this.height)):this.top??=this.virtualizer.getOffsetInScrollContainer(e),!this.isVisible)return this.renderPlaceholder(this.height);let o=this.virtualizer.getWindowSpecs(),s=this.computeRenderRangeFromWindow(this.fileDiff,this.top,o);return super.render({fileDiff:this.fileDiff,fileContainer:e,renderRange:s,oldFile:t,newFile:n,...i})}getDiffStyle(){return this.options.diffStyle??`split`}getExpandedRegion(e,t,n){if(n<=0||e)return{fromStart:0,fromEnd:0,collapsedLines:Math.max(n,0),renderAll:!1};let{expandUnchanged:r=!1,collapsedContextThreshold:i=1}=this.options;if(r||n<=i)return{fromStart:n,fromEnd:0,collapsedLines:0,renderAll:!0};let a=this.hunksRenderer.getExpandedHunk(t),o=Math.min(Math.max(a.fromStart,0),n),s=Math.min(Math.max(a.fromEnd,0),n),c=o+s,l=c>=n;return{fromStart:o,fromEnd:s,collapsedLines:Math.max(n-c,0),renderAll:l}}getExpandedLineCount(e,t){let n=0;if(e.isPartial){for(let r of e.hunks)n+=t===`split`?r.splitLineCount:r.unifiedLineCount;return n}for(let[r,i]of e.hunks.entries()){let a=t===`split`?i.splitLineCount:i.unifiedLineCount;n+=a;let o=Math.max(i.collapsedBefore,0),{fromStart:s,fromEnd:c,renderAll:l}=this.getExpandedRegion(e.isPartial,r,o);o>0&&(n+=l?o:s+c)}let r=e.hunks.at(-1);if(r!=null&&$n(e)){let t=e.additionLines.length-(r.additionLineIndex+r.additionCount),i=e.deletionLines.length-(r.deletionLineIndex+r.deletionCount);if(r!=null&&t!==i)throw Error(`VirtualizedFileDiff: trailing context mismatch (additions=${t}, deletions=${i}) for ${e.name}`);let a=Math.min(t,i);if(r!=null&&a>0){let{fromStart:t,renderAll:r}=this.getExpandedRegion(e.isPartial,e.hunks.length,a);n+=r?a:t}}return n}computeRenderRangeFromWindow(e,t,{top:n,bottom:r}){let{disableFileHeader:i=!1,expandUnchanged:a=!1,collapsedContextThreshold:o=1,hunkSeparators:s=`line-info`}=this.options,{diffHeaderHeight:c,fileGap:l,hunkLineCount:u,hunkSeparatorHeight:d,lineHeight:f}=this.metrics,p=this.getDiffStyle(),m=this.height,h=this.getExpandedLineCount(e,p),g=i?l:c;if(t<n-m||t>r)return{startingLine:0,totalLines:0,bufferBefore:0,bufferAfter:m-g-l};if(h<=u||e.hunks.length===0)return{startingLine:0,totalLines:u,bufferBefore:0,bufferAfter:0};let _=Math.ceil(Math.max(r-n,0)/f),v=Math.ceil(_/u)*u+u,y=v/u,b=y,x=[],S=(n+r)/2,C=s===`simple`||s===`metadata`||s===`line-info-basic`?0:l,w=t+g,T=0,E,D,O;if(Ge({diff:e,diffStyle:p,expandedHunks:a?!0:this.hunksRenderer.getExpandedHunksMap(),collapsedContextThreshold:o,callback:({hunkIndex:e,collapsedBefore:i,collapsedAfter:a,deletionLine:o,additionLine:c})=>{let l=c==null?o.splitLineIndex:c.splitLineIndex,f=c==null?o.unifiedLineIndex:c.unifiedLineIndex,m=(c?.noEOFCR??!1)||(o?.noEOFCR??!1),h=i>0?d+C+(e>0?C:0):0;e===0&&s===`simple`&&(h=0),w+=h;let _=T%u===0;if(_&&(x.push(w-(t+g+h)),O!=null)){if(O<=0)return!0;O--}let v=this.getLineHeight(p===`split`?l:f,m),y=Math.floor(T/u);return w>n-v&&w<r&&(E??=y),D==null&&w+v>S&&(D=y),O==null&&w>=r&&_&&(O=b),T++,w+=v,a>0&&s!==`simple`&&(w+=d+C),!1}}),E==null)return{startingLine:0,totalLines:0,bufferBefore:0,bufferAfter:m-g-l};let k=x.length;D??=E;let A=Math.round(D-y/2),j=Math.max(0,k-y),M=Math.max(0,Math.min(A,j)),N=M*u,P=A<0?v+A*u:v,F=x[M]??0,I=M+P/u;return{startingLine:N,totalLines:P,bufferBefore:F,bufferAfter:I<x.length?m-g-x[I]-l:m-(w-t)-l}}};function $n(e){let t=e.hunks.at(-1);return t==null||e.isPartial||e.additionLines.length===0||e.deletionLines.length===0?!1:t.additionLineIndex+t.additionCount<e.additionLines.length||t.deletionLineIndex+t.deletionCount<e.deletionLines.length}function er(e,t){let[n,r]=e.split(`,`).map(Number);return t===`split`?r:n}function tr(e,t,n){if(e===t||e==null||t==null)return e===t;let r=new Set(n),i=Object.keys(e),a=new Set(Object.keys(t));for(let n of i)if(a.delete(n),!r.has(n)&&(!(n in t)||e[n]!==t[n]))return!1;for(let e of Array.from(a))if(!r.has(e))return!1;return!0}function nr(e,t){return at(e?.theme??st,t?.theme??st)&&tr(e,t,[`theme`])}function rr(){let e=(0,Y.c)(3),[t,n]=a(ne),r;return e[0]!==t||e[1]!==n?(r={enterBehavior:t,setEnterBehavior:n,isLoading:!1},e[0]=t,e[1]=n,e[2]=r):r=e[2],r}var ir=`en-US`,ar=Object.assign({"../locales/am.json":()=>n(()=>import(`./am-uZiF2JpI.js`),[],import.meta.url),"../locales/ar.json":()=>n(()=>import(`./ar-B0_uIVt7.js`),[],import.meta.url),"../locales/bg-BG.json":()=>n(()=>import(`./bg-BG-BQo5D-8g.js`),[],import.meta.url),"../locales/bn-BD.json":()=>n(()=>import(`./bn-BD-C_yQBGFs.js`),[],import.meta.url),"../locales/bs-BA.json":()=>n(()=>import(`./bs-BA-DjHSyHNR.js`),[],import.meta.url),"../locales/ca-ES.json":()=>n(()=>import(`./ca-ES-Bmvac4b1.js`),[],import.meta.url),"../locales/cs-CZ.json":()=>n(()=>import(`./cs-CZ-DuDrcseh.js`),[],import.meta.url),"../locales/da-DK.json":()=>n(()=>import(`./da-DK-DcqTiJ1K.js`),[],import.meta.url),"../locales/de-DE.json":()=>n(()=>import(`./de-DE-DCQleEWW.js`),[],import.meta.url),"../locales/el-GR.json":()=>n(()=>import(`./el-GR-w1xmZe8H.js`),[],import.meta.url),"../locales/es-419.json":()=>n(()=>import(`./es-419-BT2slJrr.js`),[],import.meta.url),"../locales/es-ES.json":()=>n(()=>import(`./es-ES-BlueS9Ux.js`),[],import.meta.url),"../locales/et-EE.json":()=>n(()=>import(`./et-EE-BeJIS97L.js`),[],import.meta.url),"../locales/fa.json":()=>n(()=>import(`./fa-Dm_ajPYE.js`),[],import.meta.url),"../locales/fi-FI.json":()=>n(()=>import(`./fi-FI-Yr1_jynb.js`),[],import.meta.url),"../locales/fr-CA.json":()=>n(()=>import(`./fr-CA-Bu5F3Zhi.js`),[],import.meta.url),"../locales/fr-FR.json":()=>n(()=>import(`./fr-FR-dzf49wD1.js`),[],import.meta.url),"../locales/gu-IN.json":()=>n(()=>import(`./gu-IN-CqvGstcn.js`),[],import.meta.url),"../locales/hi-IN.json":()=>n(()=>import(`./hi-IN-BNEGod_k.js`),[],import.meta.url),"../locales/hr-HR.json":()=>n(()=>import(`./hr-HR-DVAKkYn6.js`),[],import.meta.url),"../locales/hu-HU.json":()=>n(()=>import(`./hu-HU-DO95BpKH.js`),[],import.meta.url),"../locales/hy-AM.json":()=>n(()=>import(`./hy-AM-Ko7pTCvy.js`),[],import.meta.url),"../locales/id-ID.json":()=>n(()=>import(`./id-ID-DTJ3E1F2.js`),[],import.meta.url),"../locales/is-IS.json":()=>n(()=>import(`./is-IS-oOuNofpI.js`),[],import.meta.url),"../locales/it-IT.json":()=>n(()=>import(`./it-IT-0EuCGKXd.js`),[],import.meta.url),"../locales/ja-JP.json":()=>n(()=>import(`./ja-JP-Da90d28E.js`),[],import.meta.url),"../locales/ka-GE.json":()=>n(()=>import(`./ka-GE-DAEp-R9B.js`),[],import.meta.url),"../locales/kk.json":()=>n(()=>import(`./kk-Cq4WrIUd.js`),[],import.meta.url),"../locales/kn-IN.json":()=>n(()=>import(`./kn-IN-CbQWt7be.js`),[],import.meta.url),"../locales/ko-KR.json":()=>n(()=>import(`./ko-KR-CZAVb2Y9.js`),[],import.meta.url),"../locales/lt.json":()=>n(()=>import(`./lt-Czf-pxgz.js`),[],import.meta.url),"../locales/lv-LV.json":()=>n(()=>import(`./lv-LV-BixECqO4.js`),[],import.meta.url),"../locales/mk-MK.json":()=>n(()=>import(`./mk-MK-Cj71UfJx.js`),[],import.meta.url),"../locales/ml.json":()=>n(()=>import(`./ml-DslGTHKz.js`),[],import.meta.url),"../locales/mn.json":()=>n(()=>import(`./mn-Chm4v4uk.js`),[],import.meta.url),"../locales/mr-IN.json":()=>n(()=>import(`./mr-IN-CBzKc0Hj.js`),[],import.meta.url),"../locales/ms-MY.json":()=>n(()=>import(`./ms-MY-DymbF0NJ.js`),[],import.meta.url),"../locales/my-MM.json":()=>n(()=>import(`./my-MM-4guCYIB3.js`),[],import.meta.url),"../locales/nb-NO.json":()=>n(()=>import(`./nb-NO-MGui16lj.js`),[],import.meta.url),"../locales/nl-NL.json":()=>n(()=>import(`./nl-NL-DMp4UfJo.js`),[],import.meta.url),"../locales/pa.json":()=>n(()=>import(`./pa-Bw-8ClKq.js`),[],import.meta.url),"../locales/pl-PL.json":()=>n(()=>import(`./pl-PL-CUlUwzCz.js`),[],import.meta.url),"../locales/pt-BR.json":()=>n(()=>import(`./pt-BR-ghf1PxM2.js`),[],import.meta.url),"../locales/pt-PT.json":()=>n(()=>import(`./pt-PT-3SUpw7-7.js`),[],import.meta.url),"../locales/ro-RO.json":()=>n(()=>import(`./ro-RO-CMoBhM6s.js`),[],import.meta.url),"../locales/ru-RU.json":()=>n(()=>import(`./ru-RU-C5rC1uaJ.js`),[],import.meta.url),"../locales/sk-SK.json":()=>n(()=>import(`./sk-SK-BCQzMC6a.js`),[],import.meta.url),"../locales/sl-SI.json":()=>n(()=>import(`./sl-SI-aNZKiQuE.js`),[],import.meta.url),"../locales/so-SO.json":()=>n(()=>import(`./so-SO-CUtcRKUU.js`),[],import.meta.url),"../locales/sq-AL.json":()=>n(()=>import(`./sq-AL-DeceBDM-.js`),[],import.meta.url),"../locales/sr-RS.json":()=>n(()=>import(`./sr-RS-CGRXXf_q.js`),[],import.meta.url),"../locales/sv-SE.json":()=>n(()=>import(`./sv-SE-MsqAGmfO.js`),[],import.meta.url),"../locales/sw-TZ.json":()=>n(()=>import(`./sw-TZ-D3k8oknF.js`),[],import.meta.url),"../locales/ta-IN.json":()=>n(()=>import(`./ta-IN-zqBFTpWT.js`),[],import.meta.url),"../locales/te-IN.json":()=>n(()=>import(`./te-IN-DAvpvO-e.js`),[],import.meta.url),"../locales/th-TH.json":()=>n(()=>import(`./th-TH-Bv__AhTB.js`),[],import.meta.url),"../locales/tl.json":()=>n(()=>import(`./tl-EoV9VIwD.js`),[],import.meta.url),"../locales/tr-TR.json":()=>n(()=>import(`./tr-TR-D3Hl8ABj.js`),[],import.meta.url),"../locales/uk-UA.json":()=>n(()=>import(`./uk-UA-BZrbZUXW.js`),[],import.meta.url),"../locales/ur.json":()=>n(()=>import(`./ur-CkYnZtKj.js`),[],import.meta.url),"../locales/vi-VN.json":()=>n(()=>import(`./vi-VN-unaxvDBL.js`),[],import.meta.url),"../locales/zh-CN.json":()=>n(()=>import(`./zh-CN-0CnfFsLt.js`),[],import.meta.url),"../locales/zh-HK.json":()=>n(()=>import(`./zh-HK-BSSFX_hr.js`),[],import.meta.url),"../locales/zh-TW.json":()=>n(()=>import(`./zh-TW-B8SATe9N.js`),[],import.meta.url)}),or=Object.entries(ar).map(([e,t])=>{let n=e.match(/\/([^/]+)\.json$/);if(!n)return null;let r=n[1],i=cr(r);if(!i)return null;let[a]=i.split(`-`);return{locale:r,normalized:i,language:a,load:t}}).filter(e=>e!=null).sort((e,t)=>e.locale.localeCompare(t.locale));function sr(){return[...or]}function cr(e){if(!e)return null;let t=e.trim();return t?t.replace(/_/g,`-`).toLowerCase():null}function lr(e){let t=cr(e);return t==null?!1:t===`en`||t.startsWith(`en-`)}function ur(e,t){return lr(e)?lr(t):cr(e)===cr(t)}function dr(e){let t=cr(e);if(!t)return;let n=or.find(e=>e.normalized===t);if(n)return n;let[r,i]=t.split(`-`);if(!r)return;let a=or.filter(e=>e.language===r);if(a.length!==0){if(i){let e=a.find(e=>e.normalized===`${r}-${i}`);if(e)return e}return a[0]}}async function fr(e){let t=await e.load();return t.default??t}var Z=e(t(),1);function pr(){let e=(0,Y.c)(9),{data:t,setData:n,isLoading:r}=K(h.FOLLOW_UP_QUEUE_MODE),i=t===`interrupt`?`steer`:t??`queue`,a,o;e[0]!==t||e[1]!==n?(a=()=>{t===`interrupt`&&n(`steer`)},o=[t,n],e[0]=t,e[1]=n,e[2]=a,e[3]=o):(a=e[2],o=e[3]),(0,Z.useEffect)(a,o);let s=i===`queue`,c;return e[4]!==r||e[5]!==i||e[6]!==n||e[7]!==s?(c={mode:i,isQueueingEnabled:s,setMode:n,isLoading:r},e[4]=r,e[5]=i,e[6]=n,e[7]=s,e[8]=c):c=e[8],c}var Q=M(),mr=e=>(0,Q.jsx)(`svg`,{width:20,height:20,viewBox:`0 0 20 20`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`,...e,children:(0,Q.jsx)(`path`,{d:`M17.6682 13.998H12.6565L11.9641 14.3447C11.8718 14.3909 11.7695 14.415 11.6663 14.415H8.33325C8.23001 14.415 8.12774 14.3909 8.0354 14.3447L7.34302 13.998H2.32837V14.583C2.32837 15.1362 2.77712 15.585 3.33032 15.585H16.6663C17.2195 15.585 17.6682 15.1362 17.6682 14.583V13.998ZM16.8352 6.41699C16.8352 5.93931 16.8347 5.62054 16.8147 5.37598C16.8002 5.19841 16.7766 5.09313 16.7512 5.02246L16.7258 4.96191C16.6538 4.82049 16.5493 4.69891 16.4221 4.60645L16.2883 4.52441C16.2194 4.48931 16.1101 4.45489 15.8733 4.43555C15.6288 4.4156 15.3106 4.41504 14.8333 4.41504H5.16626C4.68886 4.41504 4.37071 4.41559 4.12622 4.43555C3.94903 4.45002 3.84339 4.47277 3.77271 4.49805L3.71216 4.52441C3.57094 4.59637 3.4491 4.70021 3.35669 4.82715L3.27368 4.96191C3.23861 5.03079 3.20513 5.13947 3.18579 5.37598C3.16581 5.62054 3.16528 5.93931 3.16528 6.41699V12.668H7.50024L7.57642 12.6729C7.65302 12.6817 7.72779 12.7036 7.79712 12.7383L8.4895 13.085H11.51L12.2024 12.7383L12.2737 12.708C12.346 12.6819 12.423 12.668 12.5002 12.668H16.8352V6.41699ZM18.1653 12.668H18.3333C18.7003 12.668 18.9981 12.9659 18.9983 13.333V14.583C18.9983 15.8708 17.954 16.915 16.6663 16.915H3.33032C2.04258 16.915 0.998291 15.8708 0.998291 14.583V13.333L1.01196 13.1992C1.07402 12.8962 1.34201 12.668 1.66333 12.668H1.83521V6.41699C1.83521 5.96125 1.83419 5.57886 1.85962 5.26758C1.88569 4.94869 1.94266 4.6459 2.08911 4.3584L2.17896 4.19727C2.40296 3.83215 2.72389 3.53443 3.10767 3.33887L3.21606 3.28809C3.47122 3.17862 3.73854 3.13317 4.01782 3.11035C4.32903 3.08493 4.71068 3.08496 5.16626 3.08496H14.8333C15.2888 3.08496 15.6705 3.08494 15.9817 3.11035C16.3007 3.13642 16.6042 3.19231 16.8918 3.33887L17.052 3.42871C17.4174 3.65275 17.7147 3.97437 17.9104 4.3584L17.9612 4.4668C18.0705 4.72179 18.1171 4.9885 18.1399 5.26758C18.1653 5.57886 18.1653 5.96125 18.1653 6.41699V12.668Z`,fill:`currentColor`})}),hr={position:`absolute`,top:0,bottom:0,textAlign:`center`},gr={display:`contents`};function _r(){return null}function vr(e,t){return typeof window>`u`&&t!=null?(0,Q.jsxs)(Q.Fragment,{children:[(0,Q.jsx)(`template`,{shadowrootmode:`open`,dangerouslySetInnerHTML:{__html:t}}),e]}):(0,Q.jsx)(Q.Fragment,{children:e})}var yr=(0,Z.createContext)(void 0);function br(){return(0,Z.useContext)(yr)}function xr(e){let t=(0,Z.useRef)(e);return(0,Z.useInsertionEffect)(()=>void(t.current=e)),(0,Z.useCallback)((...e)=>t.current(...e),[])}function Sr({fileDiff:e,actions:t,deletionFile:n,additionFile:r,renderHeaderPrefix:i,renderHeaderMetadata:a,renderAnnotation:o,renderGutterUtility:s,renderHoverUtility:c,renderMergeConflictUtility:l,lineAnnotations:u,getHoveredLine:d,getInstance:f}){let p=s??c,m=i?.({fileDiff:e,deletionFile:n,additionFile:r}),h=a?.({fileDiff:e,deletionFile:n,additionFile:r});return(0,Q.jsxs)(Q.Fragment,{children:[m!=null&&(0,Q.jsx)(`div`,{slot:`header-prefix`,children:m}),h!=null&&(0,Q.jsx)(`div`,{slot:`header-metadata`,children:h}),o!=null&&u?.map((e,t)=>(0,Q.jsx)(`div`,{slot:nn(e),children:o(e)},t)),t!=null&&l!=null&&f!=null&&t.map(e=>{let t=Cr(e);return(0,Q.jsx)(`div`,{slot:t,style:gr,children:l(e,f)},t)}),p!=null&&(0,Q.jsx)(`div`,{slot:`gutter-utility-slot`,style:hr,children:p(d)})]})}function Cr(e){let t=Jn(e);return t==null?void 0:qn({...t,conflictIndex:e.conflictIndex})}var wr=typeof window>`u`?Z.useEffect:Z.useLayoutEffect;function Tr({oldFile:e,newFile:t,fileDiff:n,options:r,lineAnnotations:i,selectedLines:a,prerenderedHTML:o,metrics:s,hasGutterRenderUtility:c}){let l=br(),u=(0,Z.useContext)(We),d=(0,Z.useRef)(null),f=xr(a=>{if(a!=null){if(d.current!=null)throw Error(`useFileDiffInstance: An instance should not already exist when a node is created`);l==null?d.current=new Gn(Er(r,c),u,!0):d.current=new Qn(Er(r,c),l,s,u,!0),d.current.hydrate({fileDiff:n,oldFile:e,newFile:t,fileContainer:a,lineAnnotations:i,prerenderedHTML:o})}else{if(d.current==null)throw Error(`useFileDiffInstance: A FileDiff instance should exist when unmounting`);d.current.cleanUp(),d.current=null}});return wr(()=>{let{current:o}=d;if(o==null)return;let s=Er(r,c),l=!nr(o.options,s);o.setOptions(s),o.render({forceRender:l,fileDiff:n,oldFile:e,newFile:t,lineAnnotations:i}),a!==void 0&&o.setSelectedLines(a)}),{ref:f,getHoveredLine:(0,Z.useCallback)(()=>d.current?.getHoveredLine(),[])}}function Er(e,t){return t?{...e,renderGutterUtility:_r}:e}function Dr({fileDiff:e,options:t,metrics:n,lineAnnotations:r,selectedLines:i,className:a,style:o,prerenderedHTML:s,renderAnnotation:c,renderHeaderPrefix:l,renderHeaderMetadata:u,renderGutterUtility:d,renderHoverUtility:f}){let{ref:p,getHoveredLine:m}=Tr({fileDiff:e,options:t,metrics:n,lineAnnotations:r,selectedLines:i,prerenderedHTML:s,hasGutterRenderUtility:d!=null||f!=null});return(0,Q.jsx)(Ie,{ref:p,className:a,style:o,children:vr(Sr({fileDiff:e,renderHeaderPrefix:l,renderHeaderMetadata:u,renderAnnotation:c,renderGutterUtility:d,lineAnnotations:r,renderHoverUtility:f,getHoveredLine:m}),s)})}var Or=`color-mix(
  in srgb,
  var(--color-token-side-bar-background) 97%,
  var(--color-token-foreground)
)`,kr=`
  --codex-diffs-surface: ${Or};
  --codex-diffs-context-number: color-mix(
  in lab,
  var(--codex-diffs-surface) 98.5%,
  var(--diffs-mixer)
);
  --codex-diffs-addition-number: color-mix(
  in srgb,
  var(--codex-diffs-surface) 91%,
  var(--diffs-addition-color-override)
);
  --codex-diffs-deletion-number: color-mix(
  in srgb,
  var(--codex-diffs-surface) 91%,
  var(--diffs-deletion-color-override)
);
`;function Ar(e){return e?Or:`var(--color-token-side-bar-background)`}var jr=`:is([data-diff], [data-file])`;function Mr(e){let t=(0,Y.c)(52),n,r,i,a,o,s,c,l,u;t[0]===e?(n=t[1],r=t[2],i=t[3],a=t[4],o=t[5],s=t[6],c=t[7],l=t[8],u=t[9]):({fileDiff:r,className:n,hunkSeparators:c,isLoadingFullContent:l,lineAnnotations:i,onGutterUtilityClick:a,renderAnnotation:o,overflow:u,...s}=e,t[0]=e,t[1]=n,t[2]=r,t[3]=i,t[4]=a,t[5]=o,t[6]=s,t[7]=c,t[8]=l,t[9]=u);let d=c===void 0?`line-info`:c,f=u===void 0?`scroll`:u,p=z(),m=$e(Xe()),g=p===`electron`,_;t[10]===g?_=t[11]:(_={enabled:g},t[10]=g,t[11]=_);let{data:v}=K(h.APPEARANCE_LIGHT_CODE_THEME_ID,_),y=p===`electron`,b;t[12]===y?b=t[13]:(b={enabled:y},t[12]=y,t[13]=b);let{data:x}=K(h.APPEARANCE_DARK_CODE_THEME_ID,b),S=j(),C;t[14]!==m||t[15]!==x||t[16]!==v?(C=m===`light`?et(v,`light`):et(x,`dark`),t[14]=m,t[15]=x,t[16]=v,t[17]=C):C=t[17];let w=C,T,E,D,O,k,A,M,N,P;if(t[18]!==m||t[19]!==n||t[20]!==r||t[21]!==d||t[22]!==S||t[23]!==i||t[24]!==f||t[25]!==o||t[26]!==w||t[27]!==p){let e=He(void 0);T=Dr,k=n,A=r,M=i,N=o,P=f,E=d,D=p===`extension`&&S!=null?S?`dark`:`light`:m,O=p===`extension`?{dark:et(e.id,`dark`).name,light:et(e.id,`light`).name}:w.name,t[18]=m,t[19]=n,t[20]=r,t[21]=d,t[22]=S,t[23]=i,t[24]=f,t[25]=o,t[26]=w,t[27]=p,t[28]=T,t[29]=E,t[30]=D,t[31]=O,t[32]=k,t[33]=A,t[34]=M,t[35]=N,t[36]=P}else T=t[28],E=t[29],D=t[30],O=t[31],k=t[32],A=t[33],M=t[34],N=t[35],P=t[36];let F=a!=null,I;t[37]!==a||t[38]!==s||t[39]!==E||t[40]!==D||t[41]!==O||t[42]!==F||t[43]!==P?(I={overflow:P,hunkSeparators:E,themeType:D,theme:O,disableFileHeader:!0,enableGutterUtility:F,onGutterUtilityClick:a,unsafeCSS:`
          [data-diffs-header],
          ${jr} {
            ${kr}
            --diffs-bg: var(--codex-diffs-surface) !important;
            background-color: var(--codex-diffs-surface) !important;
          }

          ${jr} [data-utility-button] {
            background-color: var(--color-token-foreground);
            color: var(--color-token-side-bar-background);
            border: none;
            border-radius: 4px;
          }

          ${jr} [data-utility-button]:hover {
            background-color: color-mix(
              in srgb,
              var(--color-token-foreground) 88%,
              var(--color-token-side-bar-background)
            );
          }

          mark.codex-thread-find-match {
            background-color: var(--vscode-charts-yellow);
            color: var(--color-token-foreground);
            border-radius: var(--radius-2xs);
            padding: 0;
            margin: 0;
            border: 0;
            font: inherit;
            line-height: inherit;
            letter-spacing: inherit;
            word-spacing: inherit;
            vertical-align: baseline;
          }

          mark.codex-thread-find-active {
            background-color: var(--vscode-charts-orange);
          }

          :host(.composer-diff-simple-line) [data-separator]:empty {
            background-color: transparent;
          }

          :host(.composer-diff-simple-line) [data-separator]:empty::after {
            content: "";
            grid-column: 2 / 3;
            align-self: center;
            margin-inline: 1ch;
            border-top: 1px solid color-mix(in srgb, var(--diffs-fg) 18%, transparent);
          }
        `,...s},t[37]=a,t[38]=s,t[39]=E,t[40]=D,t[41]=O,t[42]=F,t[43]=P,t[44]=I):I=t[44];let L;return t[45]!==T||t[46]!==k||t[47]!==A||t[48]!==M||t[49]!==N||t[50]!==I?(L=(0,Q.jsx)(T,{className:k,fileDiff:A,lineAnnotations:M,renderAnnotation:N,options:I}),t[45]=T,t[46]=k,t[47]=A,t[48]=M,t[49]=N,t[50]=I,t[51]=L):L=t[51],L}var Nr=p({sandbox:l([`elevated`,`unelevated`]).optional()}).passthrough();function Pr(e){return[`windows-sandbox`,`mode`,e.getHostId()]}function Fr(e){let t=(0,Y.c)(7),n=s(),i=e??n,a;t[0]===i?a=t[1]:(a=Pr(i),t[0]=i,t[1]=a);let o=a,c;t[2]===i?c=t[3]:(c=async()=>{try{return Lr((await i.readConfig({includeLayers:!1,cwd:null})).config.windows)}catch(e){let t=e;return r.error(`Failed to load Windows sandbox mode`,{safe:{error:String(t)},sensitive:{}}),null}},t[2]=i,t[3]=c);let l;return t[4]!==o||t[5]!==c?(l={queryKey:o,queryFn:c,staleTime:y.ONE_MINUTE},t[4]=o,t[5]=c,t[6]=l):l=t[6],S(l)}function Ir(e){let t=(0,Y.c)(16),n=s(),i=e??n,a=b(),o=dt(),c;t[0]===i?c=t[1]:(c=Pr(i),t[0]=i,t[1]=c);let l=c,u;t[2]===i?u=t[3]:(u=e=>i.batchWriteConfigValue({edits:[{keyPath:`windows.sandbox`,value:e,mergeStrategy:e==null?`replace`:`upsert`}],filePath:null,expectedVersion:null}),t[2]=i,t[3]=u);let d,f;t[4]!==a||t[5]!==l?(d=e=>{let t=a.getQueryData(l);return a.setQueryData(l,e),{previousMode:t}},f=(e,t,n)=>{r.error(`Failed to update Windows sandbox mode`,{safe:{error:String(e)},sensitive:{}}),n?.previousMode!==void 0&&a.setQueryData(l,n.previousMode)},t[4]=a,t[5]=l,t[6]=d,t[7]=f):(d=t[6],f=t[7]);let p;t[8]!==o||t[9]!==l?(p=async()=>{await o(l)},t[8]=o,t[9]=l,t[10]=p):p=t[10];let m;return t[11]!==u||t[12]!==d||t[13]!==f||t[14]!==p?(m={mutationFn:u,onMutate:d,onError:f,onSettled:p},t[11]=u,t[12]=d,t[13]=f,t[14]=p,t[15]=m):m=t[15],T(m)}function Lr(e){let t=Nr.safeParse(e);return t.success?t.data.sandbox??null:null}var Rr=`realtime_conversation`,zr=`2380644311`,Br=`2106641128`;function Vr(){let e=(0,Y.c)(6),t=B(Br),n=B(zr),{data:r}=de(),i;e[0]===r?i=e[1]:(i=r===void 0?[]:r,e[0]=r,e[1]=i);let a=i,o;return e[2]!==a||e[3]!==t||e[4]!==n?(o=t&&n&&a.some(Hr),e[2]=a,e[3]=t,e[4]=n,e[5]=o):o=e[5],o}function Hr(e){return e.name===`realtime_conversation`&&e.enabled}var Ur=[null,`fast`],Wr=e=>(0,Q.jsx)(`svg`,{width:20,height:20,viewBox:`0 0 20 20`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`,...e,children:(0,Q.jsx)(`path`,{d:`M8.50195 5.83319C8.50197 4.93054 8.65078 4.06203 8.92188 3.24921C5.65928 3.76613 3.16504 6.59214 3.16504 10.0002C3.16514 13.775 6.2252 16.8351 10 16.8351C12.3126 16.8351 14.3565 15.6856 15.5938 13.926C11.5915 13.4005 8.50195 9.9788 8.50195 5.83319ZM9.83203 5.83319C9.83203 9.60806 12.8921 12.6682 16.667 12.6682C16.6833 12.6682 16.6996 12.6683 16.7158 12.6682C16.9467 12.6665 17.1618 12.7849 17.2842 12.9807C17.3913 13.1521 17.4145 13.3617 17.3496 13.55L17.3164 13.6291C15.9812 16.3161 13.2069 18.1652 10 18.1652C5.49066 18.1652 1.83506 14.5095 1.83496 10.0002C1.83496 5.51033 5.45891 1.8667 9.94141 1.83514L10.0273 1.84003C10.2248 1.86428 10.4027 1.97644 10.5098 2.14764C10.6321 2.34353 10.6447 2.58923 10.542 2.79608C10.0877 3.71023 9.83205 4.74091 9.83203 5.83319Z`,fill:`currentColor`})}),Gr=e=>(0,Q.jsx)(`svg`,{width:20,height:20,viewBox:`0 0 20 20`,fill:`none`,xmlns:`http://www.w3.org/2000/svg`,...e,children:(0,Q.jsx)(`path`,{d:`M9.33447 18.3336V16.6666C9.33447 16.2995 9.63239 16.0018 9.99951 16.0016C10.3668 16.0016 10.6646 16.2994 10.6646 16.6666V18.3336C10.6644 18.7007 10.3667 18.9987 9.99951 18.9987C9.6325 18.9985 9.33465 18.7006 9.33447 18.3336ZM5.28564 14.7145L5.75635 15.1842L4.57764 16.3629C4.31799 16.6225 3.89691 16.6224 3.63721 16.3629C3.37752 16.1032 3.37753 15.6822 3.63721 15.4225L4.81592 14.2438L5.28564 14.7145ZM16.3628 15.4225C16.6223 15.6822 16.6224 16.1033 16.3628 16.3629C16.1032 16.6226 15.6821 16.6224 15.4224 16.3629L16.3628 15.4225ZM16.3628 15.4225L15.8921 15.8922L15.4224 16.3629L14.2437 15.1842L14.7144 14.7145L15.1841 14.2438L16.3628 15.4225ZM4.81592 14.2438C5.07563 13.9843 5.49671 13.9841 5.75635 14.2438C6.01582 14.5034 6.01581 14.9245 5.75635 15.1842L4.81592 14.2438ZM14.2437 14.2438C14.5033 13.9841 14.9244 13.9841 15.1841 14.2438L14.2437 15.1842C13.984 14.9245 13.984 14.5035 14.2437 14.2438ZM12.6685 9.99963C12.6683 8.5261 11.4731 7.33167 9.99951 7.33167C8.52609 7.33184 7.33172 8.52621 7.33154 9.99963C7.33154 11.4732 8.52598 12.6684 9.99951 12.6686C11.4732 12.6686 12.6685 11.4733 12.6685 9.99963ZM3.3335 9.33459L3.46729 9.34827C3.77019 9.41027 3.99838 9.67844 3.99854 9.99963C3.99854 10.3209 3.77023 10.5889 3.46729 10.651L3.3335 10.6647H1.6665C1.29923 10.6647 1.00146 10.3669 1.00146 9.99963C1.00164 9.63251 1.29934 9.33459 1.6665 9.33459H3.3335ZM18.3335 9.33459L18.4673 9.34827C18.7702 9.41027 18.9984 9.67844 18.9985 9.99963C18.9985 10.3209 18.7702 10.5889 18.4673 10.651L18.3335 10.6647H16.6665C16.2992 10.6647 16.0015 10.3669 16.0015 9.99963C16.0016 9.63251 16.2993 9.33459 16.6665 9.33459H18.3335ZM5.75635 4.81604C6.01571 5.07577 6.01593 5.49688 5.75635 5.75647C5.49676 6.01605 5.07564 6.01583 4.81592 5.75647L5.75635 4.81604ZM15.1841 5.75647C14.9244 6.01594 14.5033 6.01595 14.2437 5.75647C13.984 5.49683 13.9841 5.07575 14.2437 4.81604L15.1841 5.75647ZM3.63721 3.63733C3.86449 3.41005 4.21501 3.38183 4.47314 3.55237L4.57764 3.63733L5.75635 4.81604L5.28564 5.28577L4.81592 5.75647L3.63721 4.57776L3.55225 4.47327C3.3817 4.21513 3.40992 3.86461 3.63721 3.63733ZM15.4224 3.63733C15.6821 3.37765 16.1031 3.37764 16.3628 3.63733C16.6223 3.89703 16.6224 4.31811 16.3628 4.57776L15.1841 5.75647L14.7144 5.28577L14.2437 4.81604L15.4224 3.63733ZM9.33447 3.33362V1.66663C9.33447 1.29947 9.63239 1.00176 9.99951 1.00159C10.3668 1.00159 10.6646 1.29936 10.6646 1.66663V3.33362C10.6644 3.70074 10.3667 3.99866 9.99951 3.99866C9.6325 3.99848 9.33465 3.70063 9.33447 3.33362ZM13.9985 9.99963C13.9985 12.2079 12.2077 13.9987 9.99951 13.9987C7.79144 13.9985 6.00146 12.2077 6.00146 9.99963C6.00164 7.79167 7.79155 6.00176 9.99951 6.00159C12.2076 6.00159 13.9984 7.79156 13.9985 9.99963Z`,fill:`currentColor`})});function Kr(){return(Kr=Object.assign||function(e){for(var t=1;t<arguments.length;t++){var n=arguments[t];for(var r in n)Object.prototype.hasOwnProperty.call(n,r)&&(e[r]=n[r])}return e}).apply(this,arguments)}function qr(e,t){if(e==null)return{};var n,r,i={},a=Object.keys(e);for(r=0;r<a.length;r++)t.indexOf(n=a[r])>=0||(i[n]=e[n]);return i}function Jr(e){var t=(0,Z.useRef)(e),n=(0,Z.useRef)(function(e){t.current&&t.current(e)});return t.current=e,n.current}var Yr=function(e,t,n){return t===void 0&&(t=0),n===void 0&&(n=1),e>n?n:e<t?t:e},Xr=function(e){return`touches`in e},Zr=function(e){return e&&e.ownerDocument.defaultView||self},Qr=function(e,t,n){var r=e.getBoundingClientRect(),i=Xr(t)?function(e,t){for(var n=0;n<e.length;n++)if(e[n].identifier===t)return e[n];return e[0]}(t.touches,n):t;return{left:Yr((i.pageX-(r.left+Zr(e).pageXOffset))/r.width),top:Yr((i.pageY-(r.top+Zr(e).pageYOffset))/r.height)}},$r=function(e){!Xr(e)&&e.preventDefault()},ei=Z.memo(function(e){var t=e.onMove,n=e.onKey,r=qr(e,[`onMove`,`onKey`]),i=(0,Z.useRef)(null),a=Jr(t),o=Jr(n),s=(0,Z.useRef)(null),c=(0,Z.useRef)(!1),l=(0,Z.useMemo)(function(){var e=function(e){$r(e),(Xr(e)?e.touches.length>0:e.buttons>0)&&i.current?a(Qr(i.current,e,s.current)):n(!1)},t=function(){return n(!1)};function n(n){var r=c.current,a=Zr(i.current),o=n?a.addEventListener:a.removeEventListener;o(r?`touchmove`:`mousemove`,e),o(r?`touchend`:`mouseup`,t)}return[function(e){var t=e.nativeEvent,r=i.current;if(r&&($r(t),!function(e,t){return t&&!Xr(e)}(t,c.current)&&r)){if(Xr(t)){c.current=!0;var o=t.changedTouches||[];o.length&&(s.current=o[0].identifier)}r.focus(),a(Qr(r,t,s.current)),n(!0)}},function(e){var t=e.which||e.keyCode;t<37||t>40||(e.preventDefault(),o({left:t===39?.05:t===37?-.05:0,top:t===40?.05:t===38?-.05:0}))},n]},[o,a]),u=l[0],d=l[1],f=l[2];return(0,Z.useEffect)(function(){return f},[f]),Z.createElement(`div`,Kr({},r,{onTouchStart:u,onMouseDown:u,className:`react-colorful__interactive`,ref:i,onKeyDown:d,tabIndex:0,role:`slider`}))}),ti=function(e){return e.filter(Boolean).join(` `)},ni=function(e){var t=e.color,n=e.left,r=e.top,i=r===void 0?.5:r,a=ti([`react-colorful__pointer`,e.className]);return Z.createElement(`div`,{className:a,style:{top:100*i+`%`,left:100*n+`%`}},Z.createElement(`div`,{className:`react-colorful__pointer-fill`,style:{backgroundColor:t}}))},$=function(e,t,n){return t===void 0&&(t=0),n===void 0&&(n=10**t),Math.round(n*e)/n};360/(2*Math.PI);var ri=function(e){return di(ii(e))},ii=function(e){return e[0]===`#`&&(e=e.substring(1)),e.length<6?{r:parseInt(e[0]+e[0],16),g:parseInt(e[1]+e[1],16),b:parseInt(e[2]+e[2],16),a:e.length===4?$(parseInt(e[3]+e[3],16)/255,2):1}:{r:parseInt(e.substring(0,2),16),g:parseInt(e.substring(2,4),16),b:parseInt(e.substring(4,6),16),a:e.length===8?$(parseInt(e.substring(6,8),16)/255,2):1}},ai=function(e){return ui(ci(e))},oi=function(e){var t=e.s,n=e.v,r=e.a,i=(200-t)*n/100;return{h:$(e.h),s:$(i>0&&i<200?t*n/100/(i<=100?i:200-i)*100:0),l:$(i/2),a:$(r,2)}},si=function(e){var t=oi(e);return`hsl(`+t.h+`, `+t.s+`%, `+t.l+`%)`},ci=function(e){var t=e.h,n=e.s,r=e.v,i=e.a;t=t/360*6,n/=100,r/=100;var a=Math.floor(t),o=r*(1-n),s=r*(1-(t-a)*n),c=r*(1-(1-t+a)*n),l=a%6;return{r:$(255*[r,s,o,o,c,r][l]),g:$(255*[c,r,r,s,o,o][l]),b:$(255*[o,o,c,r,r,s][l]),a:$(i,2)}},li=function(e){var t=e.toString(16);return t.length<2?`0`+t:t},ui=function(e){var t=e.r,n=e.g,r=e.b,i=e.a,a=i<1?li($(255*i)):``;return`#`+li(t)+li(n)+li(r)+a},di=function(e){var t=e.r,n=e.g,r=e.b,i=e.a,a=Math.max(t,n,r),o=a-Math.min(t,n,r),s=o?a===t?(n-r)/o:a===n?2+(r-t)/o:4+(t-n)/o:0;return{h:$(60*(s<0?s+6:s)),s:$(a?o/a*100:0),v:$(a/255*100),a:i}},fi=Z.memo(function(e){var t=e.hue,n=e.onChange,r=ti([`react-colorful__hue`,e.className]);return Z.createElement(`div`,{className:r},Z.createElement(ei,{onMove:function(e){n({h:360*e.left})},onKey:function(e){n({h:Yr(t+360*e.left,0,360)})},"aria-label":`Hue`,"aria-valuenow":$(t),"aria-valuemax":`360`,"aria-valuemin":`0`},Z.createElement(ni,{className:`react-colorful__hue-pointer`,left:t/360,color:si({h:t,s:100,v:100,a:1})})))}),pi=Z.memo(function(e){var t=e.hsva,n=e.onChange,r={backgroundColor:si({h:t.h,s:100,v:100,a:1})};return Z.createElement(`div`,{className:`react-colorful__saturation`,style:r},Z.createElement(ei,{onMove:function(e){n({s:100*e.left,v:100-100*e.top})},onKey:function(e){n({s:Yr(t.s+100*e.left,0,100),v:Yr(t.v-100*e.top,0,100)})},"aria-label":`Color`,"aria-valuetext":`Saturation `+$(t.s)+`%, Brightness `+$(t.v)+`%`},Z.createElement(ni,{className:`react-colorful__saturation-pointer`,top:1-t.v/100,left:t.s/100,color:si(t)})))}),mi=function(e,t){if(e===t)return!0;for(var n in e)if(e[n]!==t[n])return!1;return!0},hi=function(e,t){return e.toLowerCase()===t.toLowerCase()||mi(ii(e),ii(t))};function gi(e,t,n){var r=Jr(n),i=(0,Z.useState)(function(){return e.toHsva(t)}),a=i[0],o=i[1],s=(0,Z.useRef)({color:t,hsva:a});return(0,Z.useEffect)(function(){if(!e.equal(t,s.current.color)){var n=e.toHsva(t);s.current={hsva:n,color:t},o(n)}},[t,e]),(0,Z.useEffect)(function(){var t;mi(a,s.current.hsva)||e.equal(t=e.fromHsva(a),s.current.color)||(s.current={hsva:a,color:t},r(t))},[a,e,r]),[a,(0,Z.useCallback)(function(e){o(function(t){return Object.assign({},t,e)})},[])]}var _i,vi=typeof window<`u`?Z.useLayoutEffect:Z.useEffect,yi=function(){return _i||(typeof __webpack_nonce__<`u`?__webpack_nonce__:void 0)},bi=new Map,xi=function(e){vi(function(){var t=e.current?e.current.ownerDocument:document;if(t!==void 0&&!bi.has(t)){var n=t.createElement(`style`);n.innerHTML=`.react-colorful{position:relative;display:flex;flex-direction:column;width:200px;height:200px;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;cursor:default}.react-colorful__saturation{position:relative;flex-grow:1;border-color:transparent;border-bottom:12px solid #000;border-radius:8px 8px 0 0;background-image:linear-gradient(0deg,#000,transparent),linear-gradient(90deg,#fff,hsla(0,0%,100%,0))}.react-colorful__alpha-gradient,.react-colorful__pointer-fill{content:"";position:absolute;left:0;top:0;right:0;bottom:0;pointer-events:none;border-radius:inherit}.react-colorful__alpha-gradient,.react-colorful__saturation{box-shadow:inset 0 0 0 1px rgba(0,0,0,.05)}.react-colorful__alpha,.react-colorful__hue{position:relative;height:24px}.react-colorful__hue{background:linear-gradient(90deg,red 0,#ff0 17%,#0f0 33%,#0ff 50%,#00f 67%,#f0f 83%,red)}.react-colorful__last-control{border-radius:0 0 8px 8px}.react-colorful__interactive{position:absolute;left:0;top:0;right:0;bottom:0;border-radius:inherit;outline:none;touch-action:none}.react-colorful__pointer{position:absolute;z-index:1;box-sizing:border-box;width:28px;height:28px;transform:translate(-50%,-50%);background-color:#fff;border:2px solid #fff;border-radius:50%;box-shadow:0 2px 4px rgba(0,0,0,.2)}.react-colorful__interactive:focus .react-colorful__pointer{transform:translate(-50%,-50%) scale(1.1)}.react-colorful__alpha,.react-colorful__alpha-pointer{background-color:#fff;background-image:url('data:image/svg+xml;charset=utf-8,<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill-opacity=".05"><path d="M8 0h8v8H8zM0 8h8v8H0z"/></svg>')}.react-colorful__saturation-pointer{z-index:3}.react-colorful__hue-pointer{z-index:2}`,bi.set(t,n);var r=yi();r&&n.setAttribute(`nonce`,r),t.head.appendChild(n)}},[])},Si=function(e){var t=e.className,n=e.colorModel,r=e.color,i=r===void 0?n.defaultColor:r,a=e.onChange,o=qr(e,[`className`,`colorModel`,`color`,`onChange`]),s=(0,Z.useRef)(null);xi(s);var c=gi(n,i,a),l=c[0],u=c[1],d=ti([`react-colorful`,t]);return Z.createElement(`div`,Kr({},o,{ref:s,className:d}),Z.createElement(pi,{hsva:l,onChange:u}),Z.createElement(fi,{hue:l.h,onChange:u,className:`react-colorful__last-control`}))},Ci={defaultColor:`000`,toHsva:ri,fromHsva:function(e){return ai({h:e.h,s:e.s,v:e.v,a:1})},equal:hi},wi=function(e){return Z.createElement(Si,Kr({},e,{colorModel:Ci}))},Ti=`codex-theme-v1:`,Ei=f(e=>typeof e==`string`&&Ye(e)),Di=w().regex(/^#[0-9a-fA-F]{6}$/),Oi=w().nullable(),ki=p({codeThemeId:Ei,theme:p({accent:Di,contrast:Et().int().min(0).max(100),fonts:p({code:Oi,ui:Oi}),ink:Di,opaqueWindows:u(),semanticColors:p({diffAdded:Di,diffRemoved:Di,skill:Di}),surface:Di}),variant:l([`light`,`dark`])});function Ai(e){let{chromeThemeConfigurationKey:t,codeThemeConfigurationKey:n}=Mi(e),{data:r,isLoading:i}=K(t),{data:a,isLoading:o}=K(n),{setCachedData:s,writeData:c}=ft(t),{setCachedData:l,writeData:u}=ft(n),d=nt(r,e),f=Ze(e),p={codeThemeId:He(a,e).id,theme:d},m=i||o,h=(0,Z.useRef)(p),g=(0,Z.useRef)(p),_=(0,Z.useRef)(0),v=(0,Z.useRef)(Promise.resolve());_.current===0&&(h.current=p,g.current=p);let y=(0,Z.useCallback)(e=>{h.current=e,s(e.theme),l(e.codeThemeId)},[l,s]),b=(0,Z.useCallback)(async(e,t)=>{_.current+=1,y(e);let n=async()=>{try{await t()}catch(t){throw Pi(h.current,e)&&y(g.current),t}finally{--_.current}},r=v.current.then(n,n);v.current=r.catch(()=>void 0),await r},[y]),x=(0,Z.useCallback)(async e=>{await b(e,async()=>{let t=g.current;await c(e.theme);try{await u(e.codeThemeId)}catch(e){throw await c(t.theme).catch(()=>void 0),e}g.current=e})},[b,u,c]),S=(0,Z.useCallback)(async e=>{if(m)return;let t={...h.current,theme:e};await b(t,async()=>{let n=g.current;if(n.codeThemeId!==t.codeThemeId){await u(t.codeThemeId);try{await c(e)}catch(e){throw await u(n.codeThemeId).catch(()=>void 0),e}g.current=t;return}await c(e),g.current=t})},[m,b,u,c]),C=(0,Z.useCallback)(e=>{S(Ni(h.current.theme,e)).catch(()=>void 0)},[S]),w=(0,Z.useCallback)(e=>{S(Ni(h.current.theme,{fonts:e})).catch(()=>void 0)},[S]),T=(0,Z.useCallback)(async t=>{if(m)return;let n=await qe(t,e);await x({codeThemeId:t,theme:Ni(h.current.theme,n)})},[m,x,e]),E=(0,Z.useCallback)(()=>Ii({codeThemeId:h.current.codeThemeId,theme:h.current.theme,variant:e}),[e]),D=(0,Z.useCallback)(t=>{try{return ji(t,e,f),!0}catch{return!1}},[f,e]),O=(0,Z.useCallback)(async t=>{m||await x(ji(t,e,f))},[f,m,x,e]),k=h.current;return{canImportThemeString:D,codeThemes:f,exportThemeString:E,fonts:k.theme.fonts,importThemeString:O,isDisabled:m,selectedCodeTheme:He(k.codeThemeId,e),setCodeThemeId:T,setFontsPatch:w,setThemePatch:C,theme:k.theme}}function ji(e,t,n){let r=Li(e);if(r.variant!==t)throw Error(`Theme variant mismatch`);let i=n.find(e=>e.id===r.codeThemeId);if(i==null)throw Error(`Theme code theme mismatch`);return{codeThemeId:i.id,theme:nt(r.theme,t)}}function Mi(e){return e===`light`?{chromeThemeConfigurationKey:h.APPEARANCE_LIGHT_CHROME_THEME,codeThemeConfigurationKey:h.APPEARANCE_LIGHT_CODE_THEME_ID}:{chromeThemeConfigurationKey:h.APPEARANCE_DARK_CHROME_THEME,codeThemeConfigurationKey:h.APPEARANCE_DARK_CODE_THEME_ID}}function Ni(e,t){return{...e,...t,fonts:t.fonts==null?e.fonts:{...e.fonts,...t.fonts},semanticColors:t.semanticColors==null?e.semanticColors:{...e.semanticColors,...t.semanticColors}}}function Pi(e,t){return e.codeThemeId===t.codeThemeId&&Fi(e.theme,t.theme)}function Fi(e,t){return e.accent===t.accent&&e.contrast===t.contrast&&e.fonts.code===t.fonts.code&&e.fonts.ui===t.fonts.ui&&e.ink===t.ink&&e.opaqueWindows===t.opaqueWindows&&e.semanticColors.diffAdded===t.semanticColors.diffAdded&&e.semanticColors.diffRemoved===t.semanticColors.diffRemoved&&e.semanticColors.skill===t.semanticColors.skill&&e.surface===t.surface}function Ii(e){return`${Ti}${JSON.stringify(e)}`}function Li(e){let t=e.trim();if(!t.startsWith(Ti))throw Error(`Theme share string mismatch`);let n=t.slice(15),r=n.startsWith(`{`)?n:decodeURIComponent(n);return ki.parse(JSON.parse(r))}var Ri=`const themePreview: ThemeConfig = {
  surface: "sidebar",
  accent: "#2563eb",
  contrast: 42,
};
`,zi=`const themePreview: ThemeConfig = {
  surface: "sidebar-elevated",
  accent: "#0ea5e9",
  contrast: 68,
};
`,Bi=`-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`,Vi=`ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace`,Hi=Un({name:`src/theme-preview.ts`,contents:Ri},{name:`src/theme-preview.ts`,contents:zi});function Ui(){let e=(0,Y.c)(6),t=Xe(),n,r;if(e[0]!==t){let i=tt(t);n=`flex flex-col gap-2`,r=i.map(Wi),e[0]=t,e[1]=n,e[2]=r}else n=e[1],r=e[2];let i;return e[3]!==n||e[4]!==r?(i=(0,Q.jsx)(`div`,{className:n,children:r}),e[3]=n,e[4]=r,e[5]=i):i=e[5],i}function Wi(e){return(0,Q.jsx)(Ki,{variant:e},e)}function Gi(){let e=(0,Y.c)(1),t;return e[0]===Symbol.for(`react.memo_cache_sentinel`)?(t=(0,Q.jsx)(`div`,{className:`overflow-hidden rounded-xl border border-token-border bg-token-main-surface-primary`,"data-testid":`theme-preview`,children:(0,Q.jsx)(Mr,{diffStyle:`split`,expansionLineCount:8,fileDiff:Hi,hunkSeparators:`line-info`,lineDiffType:`none`,overflow:`scroll`})}),e[0]=t):t=e[0],t}function Ki({variant:e}){let t=v(E),n=F(),r=n.formatMessage({id:`settings.general.appearance.chromeTheme.accent.short`,defaultMessage:`Accent`,description:`Short label for the accent color picker`}),i=n.formatMessage({id:`settings.general.appearance.chromeTheme.surface.short`,defaultMessage:`Background`,description:`Short label for the background color picker`}),a=n.formatMessage({id:`settings.general.appearance.chromeTheme.ink.short`,defaultMessage:`Foreground`,description:`Short label for the foreground color picker`}),o=n.formatMessage({id:`settings.general.appearance.chromeTheme.contrast.short`,defaultMessage:`Contrast`,description:`Short label for the contrast slider`}),s=n.formatMessage({id:`settings.general.appearance.chromeTheme.translucentSidebar.short`,defaultMessage:`Translucent sidebar`,description:`Short label for the translucent sidebar toggle`}),{canImportThemeString:c,codeThemes:l,exportThemeString:u,fonts:d,importThemeString:f,isDisabled:p,selectedCodeTheme:m,setCodeThemeId:h,setFontsPatch:g,setThemePatch:_,theme:y}=Ai(e),b=aa(n,e),[x,S]=(0,Z.useState)(!1),[C,w]=(0,Z.useState)(``),T=[{ariaLabel:n.formatMessage({id:`settings.general.appearance.chromeTheme.accent`,defaultMessage:`{variant} accent color`,description:`Aria label for the accent color input in chrome theme settings`},{variant:b}),label:r,role:`accent`},{ariaLabel:n.formatMessage({id:`settings.general.appearance.chromeTheme.surface`,defaultMessage:`{variant} background color`,description:`Aria label for the background color input in chrome theme settings`},{variant:b}),label:i,role:`surface`},{ariaLabel:n.formatMessage({id:`settings.general.appearance.chromeTheme.ink`,defaultMessage:`{variant} ink color`,description:`Aria label for the ink color input in chrome theme settings`},{variant:b}),label:a,role:`ink`}],D=[{ariaLabel:n.formatMessage({id:`settings.general.appearance.chromeTheme.uiFontFamily`,defaultMessage:`{variant} UI font`,description:`Aria label for the UI font input in chrome theme settings`},{variant:b}),key:`ui`,label:n.formatMessage({id:`settings.general.appearance.chromeTheme.uiFontFamily.short`,defaultMessage:`UI font`,description:`Short label for the UI font input`}),placeholder:Bi},{ariaLabel:n.formatMessage({id:`settings.general.appearance.chromeTheme.codeFontFamily`,defaultMessage:`{variant} code font`,description:`Aria label for the code font input in chrome theme settings`},{variant:b}),key:`code`,label:n.formatMessage({id:`settings.general.appearance.chromeTheme.codeFontFamily.short`,defaultMessage:`Code font`,description:`Short label for the code font input`}),placeholder:Vi}],O=(e,t)=>{switch(e){case`accent`:_({accent:t});return;case`ink`:_({ink:t});return;case`surface`:_({surface:t});return}},k=async()=>{if(await ct(u()).catch(()=>!1)){t.get(P).success(n.formatMessage({id:`settings.general.appearance.chromeTheme.export.success`,defaultMessage:`{variant} theme copied`,description:`Success toast shown after copying a theme share string`},{variant:b}));return}t.get(P).danger(n.formatMessage({id:`settings.general.appearance.chromeTheme.export.error`,defaultMessage:`Couldn’t copy {variant} theme`,description:`Error toast shown when copying a theme share string fails`},{variant:b}))},A=async()=>{try{let e=C.trim();if(!e)throw Error(`Missing theme string`);await f(e),S(!1),w(``),t.get(P).success(n.formatMessage({id:`settings.general.appearance.chromeTheme.import.success`,defaultMessage:`{variant} theme imported`,description:`Success toast shown after importing a theme share string`},{variant:b}))}catch{t.get(P).danger(n.formatMessage({id:`settings.general.appearance.chromeTheme.import.error`,defaultMessage:`Couldn’t import {variant} theme`,description:`Error toast shown when importing a theme share string fails`},{variant:b}))}};return(0,Q.jsxs)(qi,{title:ia(e),headerControl:(0,Q.jsxs)(`div`,{className:`flex items-center gap-2 max-sm:w-full max-sm:flex-wrap max-sm:justify-end`,children:[(0,Q.jsx)(R,{className:`px-2`,color:`ghost`,disabled:p,size:`toolbar`,onClick:()=>{S(!0)},children:(0,Q.jsx)(I,{id:`settings.general.appearance.chromeTheme.import`,defaultMessage:`Import`,description:`Button label for importing a shared theme string`})}),(0,Q.jsx)(R,{className:`px-2`,color:`ghost`,disabled:p,size:`toolbar`,onClick:()=>{k()},children:(0,Q.jsx)(I,{id:`settings.general.appearance.chromeTheme.export`,defaultMessage:`Copy theme`,description:`Button label for copying a shared theme string`})}),(0,Q.jsx)(Yi,{ariaLabel:n.formatMessage({id:`settings.general.appearance.codeTheme`,defaultMessage:`{variant} code theme`,description:`Aria label for the code theme picker in appearance settings`},{variant:b}),codeThemes:l,disabled:p,selectedCodeTheme:m,theme:y,variant:e,onSelect:e=>{h(e).catch(()=>void 0)}})]}),children:[T.map(e=>(0,Q.jsx)(q,{control:(0,Q.jsx)(Qi,{ariaLabel:e.ariaLabel,disabled:p,value:y[e.role],onChange:t=>{O(e.role,t)}}),label:e.label,variant:`nested`},e.role)),D.map(e=>(0,Q.jsx)(q,{control:(0,Q.jsx)(na,{ariaLabel:e.ariaLabel,disabled:p,placeholder:e.placeholder,value:d[e.key],onChange:t=>{g({[e.key]:t})}}),label:e.label,variant:`nested`},e.key)),(0,Q.jsx)(q,{control:(0,Q.jsx)(yt,{checked:!y.opaqueWindows,disabled:p,onChange:e=>{_({opaqueWindows:!e})},ariaLabel:n.formatMessage({id:`settings.general.appearance.chromeTheme.translucentSidebar`,defaultMessage:`{variant} translucent sidebar`,description:`Aria label for the translucent sidebar toggle in chrome theme settings`},{variant:b})}),label:s,variant:`nested`}),(0,Q.jsx)(q,{control:(0,Q.jsx)(ra,{ariaLabel:n.formatMessage({id:`settings.general.appearance.chromeTheme.contrast`,defaultMessage:`{variant} contrast`,description:`Aria label for the contrast slider in chrome theme settings`},{variant:aa(n,e)}),disabled:p,theme:y,value:y.contrast,onChange:e=>{_({contrast:e})}}),label:o,variant:`nested`}),(0,Q.jsx)(Ji,{exampleValue:u(),isImportValueValid:c(C),isDisabled:p,isOpen:x,value:C,variantLabel:b,onOpenChange:e=>{S(e),e||w(``)},onSubmit:()=>{A()},onValueChange:w})]})}function qi(e){let t=(0,Y.c)(12),{title:n,headerControl:r,children:i}=e,a;t[0]===n?a=t[1]:(a=(0,Q.jsx)(`div`,{className:`min-w-0`,children:(0,Q.jsx)(`div`,{className:`text-[13px] font-medium text-token-text-secondary`,children:n})}),t[0]=n,t[1]=a);let o;t[2]===r?o=t[3]:(o=(0,Q.jsx)(`div`,{className:`shrink-0 max-sm:w-full`,children:r}),t[2]=r,t[3]=o);let s;t[4]!==a||t[5]!==o?(s=(0,Q.jsxs)(`div`,{className:`flex items-center justify-between gap-2 px-4 py-2 max-sm:flex-col max-sm:items-stretch`,children:[a,o]}),t[4]=a,t[5]=o,t[6]=s):s=t[6];let c;t[7]===i?c=t[8]:(c=(0,Q.jsx)(`div`,{className:`divide-y-[0.5px] divide-token-border`,children:i}),t[7]=i,t[8]=c);let l;return t[9]!==s||t[10]!==c?(l=(0,Q.jsxs)(`div`,{className:`overflow-hidden rounded-2xl border border-token-border bg-token-input-background shadow-sm`,children:[s,c]}),t[9]=s,t[10]=c,t[11]=l):l=t[11],l}function Ji(e){let t=(0,Y.c)(36),{exampleValue:n,isImportValueValid:r,isDisabled:i,isOpen:a,value:o,variantLabel:s,onOpenChange:c,onSubmit:l,onValueChange:u}=e,d=F(),f;t[0]===d?f=t[1]:(f=d.formatMessage({id:`settings.general.appearance.chromeTheme.import.dialog.title`,defaultMessage:`Import theme`,description:`Title for the theme import dialog`}),t[0]=d,t[1]=f);let p=f,m;t[2]===Symbol.for(`react.memo_cache_sentinel`)?(m={"aria-describedby":void 0},t[2]=m):m=t[2];let h;t[3]===p?h=t[4]:(h=(0,Q.jsxs)(pt,{children:[(0,Q.jsx)(oe,{asChild:!0,children:(0,Q.jsx)(`h2`,{className:`sr-only`,children:p})}),(0,Q.jsx)(mt,{title:p})]}),t[3]=p,t[4]=h);let g;t[5]!==d||t[6]!==s?(g=d.formatMessage({id:`settings.general.appearance.chromeTheme.import.dialog.ariaLabel`,defaultMessage:`{variant} theme share string`,description:`Aria label for the theme import text area`},{variant:s}),t[5]=d,t[6]=s,t[7]=g):g=t[7];let _;t[8]===u?_=t[9]:(_=e=>{u(e.target.value)},t[8]=u,t[9]=_);let v;t[10]!==n||t[11]!==i||t[12]!==g||t[13]!==_||t[14]!==o?(v=(0,Q.jsx)(pt,{children:(0,Q.jsx)(`input`,{"aria-label":g,autoFocus:!0,className:`h-9 w-full rounded-xl border border-token-input-border bg-token-input-background px-3 font-mono text-sm text-token-input-foreground outline-none placeholder:text-token-input-placeholder-foreground focus:border-token-focus-border`,disabled:i,placeholder:n,spellCheck:!1,type:`text`,value:o,onChange:_})}),t[10]=n,t[11]=i,t[12]=g,t[13]=_,t[14]=o,t[15]=v):v=t[15];let y;t[16]===c?y=t[17]:(y=()=>{c(!1)},t[16]=c,t[17]=y);let b;t[18]===Symbol.for(`react.memo_cache_sentinel`)?(b=(0,Q.jsx)(I,{id:`settings.general.appearance.chromeTheme.import.dialog.cancel`,defaultMessage:`Cancel`,description:`Button label for canceling the theme import dialog`}),t[18]=b):b=t[18];let x;t[19]===y?x=t[20]:(x=(0,Q.jsx)(R,{color:`ghost`,size:`toolbar`,onClick:y,children:b}),t[19]=y,t[20]=x);let S=i||!r,C;t[21]===Symbol.for(`react.memo_cache_sentinel`)?(C=(0,Q.jsx)(I,{id:`settings.general.appearance.chromeTheme.import.dialog.submit`,defaultMessage:`Import theme`,description:`Button label for submitting a theme import`}),t[21]=C):C=t[21];let w;t[22]!==l||t[23]!==S?(w=(0,Q.jsx)(R,{color:`primary`,disabled:S,size:`toolbar`,onClick:l,children:C}),t[22]=l,t[23]=S,t[24]=w):w=t[24];let T;t[25]!==w||t[26]!==x?(T=(0,Q.jsx)(pt,{children:(0,Q.jsxs)(gt,{className:_t,children:[x,w]})}),t[25]=w,t[26]=x,t[27]=T):T=t[27];let E;t[28]!==T||t[29]!==h||t[30]!==v?(E=(0,Q.jsxs)(ht,{children:[h,v,T]}),t[28]=T,t[29]=h,t[30]=v,t[31]=E):E=t[31];let D;return t[32]!==a||t[33]!==c||t[34]!==E?(D=(0,Q.jsx)(se,{open:a,onOpenChange:c,size:`default`,contentProps:m,children:E}),t[32]=a,t[33]=c,t[34]=E,t[35]=D):D=t[35],D}function Yi(e){let t=(0,Y.c)(20),{ariaLabel:n,codeThemes:r,disabled:i,selectedCodeTheme:a,theme:o,variant:s,onSelect:c}=e,l;t[0]===o?l=t[1]:(l=(0,Q.jsx)(Zi,{theme:o}),t[0]=o,t[1]=l);let u;t[2]===a.label?u=t[3]:(u=(0,Q.jsx)(`span`,{className:`truncate text-sm leading-none`,children:a.label}),t[2]=a.label,t[3]=u);let d;t[4]!==n||t[5]!==i||t[6]!==l||t[7]!==u?(d=(0,Q.jsxs)(wt,{"aria-label":n,className:`h-9 w-[11rem] justify-between rounded-lg border border-token-border bg-token-bg-primary px-2.5 py-0 shadow-sm max-sm:w-full`,contentClassName:`gap-2`,chevronClassName:`icon-xs opacity-65`,disabled:i,children:[l,u]}),t[4]=n,t[5]=i,t[6]=l,t[7]=u,t[8]=d):d=t[8];let f;t[9]!==r||t[10]!==i||t[11]!==c||t[12]!==a||t[13]!==o||t[14]!==s?(f=(0,Q.jsx)(Xi,{codeThemes:r,disabled:i,selectedCodeTheme:a,theme:o,variant:s,onSelect:c}),t[9]=r,t[10]=i,t[11]=c,t[12]=a,t[13]=o,t[14]=s,t[15]=f):f=t[15];let p;return t[16]!==i||t[17]!==d||t[18]!==f?(p=(0,Q.jsx)(he,{align:`end`,contentWidth:`menuWide`,disabled:i,triggerButton:d,children:f}),t[16]=i,t[17]=d,t[18]=f,t[19]=p):p=t[19],p}function Xi(e){let t=(0,Y.c)(22),{codeThemes:n,disabled:r,selectedCodeTheme:i,theme:a,variant:o,onSelect:s}=e,c;if(t[0]!==n||t[1]!==o){let e;t[3]===o?e=t[4]:(e=e=>({queryKey:[`code-theme-preview-seed`,o,e.id],queryFn:()=>qe(e.id,o),staleTime:1/0}),t[3]=o,t[4]=e),c=n.map(e),t[0]=n,t[1]=o,t[2]=c}else c=t[2];let l;t[5]===c?l=t[6]:(l={queries:c},t[5]=c,t[6]=l);let u=A(l),d;if(t[7]!==n||t[8]!==r||t[9]!==s||t[10]!==i||t[11]!==a||t[12]!==u){let e;t[14]!==r||t[15]!==s||t[16]!==i||t[17]!==a||t[18]!==u?(e=(e,t)=>{let n=u[t]?.data;return(0,Q.jsx)(U.Item,{disabled:r,RightIcon:e.id===i.id?H:void 0,onSelect:()=>{s(e.id)},children:(0,Q.jsxs)(`div`,{className:`flex items-center gap-2`,children:[(0,Q.jsx)(Zi,{theme:{accent:n?.accent??a.accent,ink:n?.ink??a.ink,surface:n?.surface??a.surface}}),(0,Q.jsx)(`span`,{className:`truncate`,children:e.label})]})},e.id)},t[14]=r,t[15]=s,t[16]=i,t[17]=a,t[18]=u,t[19]=e):e=t[19],d=n.map(e),t[7]=n,t[8]=r,t[9]=s,t[10]=i,t[11]=a,t[12]=u,t[13]=d}else d=t[13];let f;return t[20]===d?f=t[21]:(f=(0,Q.jsx)(U.Section,{children:(0,Q.jsx)(`div`,{className:`max-h-80 overflow-y-auto pb-1`,children:d})}),t[20]=d,t[21]=f),f}function Zi(e){let t=(0,Y.c)(9),{theme:n}=e,r=F(),i=`color-mix(in srgb, ${n.ink} 16%, ${n.surface})`,a;t[0]!==i||t[1]!==n.accent||t[2]!==n.surface?(a={backgroundColor:n.surface,borderColor:i,color:n.accent},t[0]=i,t[1]=n.accent,t[2]=n.surface,t[3]=a):a=t[3];let o;t[4]===r?o=t[5]:(o=r.formatMessage({id:`settings.general.appearance.codeTheme.previewGlyph`,defaultMessage:`Aa`,description:`Preview glyph shown in the code theme selector`}),t[4]=r,t[5]=o);let s;return t[6]!==a||t[7]!==o?(s=(0,Q.jsx)(`span`,{"aria-hidden":!0,className:`flex h-6 w-6 shrink-0 items-center justify-center rounded-md border text-[11px] leading-none font-semibold`,style:a,children:o}),t[6]=a,t[7]=o,t[8]=s):s=t[8],s}function Qi(e){let t=(0,Y.c)(38),{ariaLabel:n,disabled:r,value:i,onChange:a}=e,[o,s]=(0,Z.useState)(!1),[c,l]=(0,Z.useState)(null),u;t[0]===Symbol.for(`react.memo_cache_sentinel`)?(u=e=>{s(e),e||l(null)},t[0]=u):u=t[0];let d;t[1]===i?d=t[2]:(d=oa(i),t[1]=i,t[2]=d);let f;t[3]!==d||t[4]!==i?(f={backgroundColor:i,color:d},t[3]=d,t[4]=i,t[5]=f):f=t[5];let p;t[6]===i?p=t[7]:(p=oa(i),t[6]=i,t[7]=p);let m=`1px solid color-mix(in srgb, ${p} 18%, ${i})`,h;t[8]!==m||t[9]!==i?(h={backgroundColor:i,border:m},t[8]=m,t[9]=i,t[10]=h):h=t[10];let g;t[11]===Symbol.for(`react.memo_cache_sentinel`)?(g=(0,Q.jsx)(`span`,{"aria-hidden":!0,className:`sr-only`}),t[11]=g):g=t[11];let _;t[12]!==r||t[13]!==h?(_=(0,Q.jsx)(bt,{asChild:!0,children:(0,Q.jsx)(`button`,{className:`h-3.5 w-3.5 shrink-0 rounded-full disabled:cursor-default`,disabled:r,style:h,type:`button`,children:g})}),t[12]=r,t[13]=h,t[14]=_):_=t[14];let v=c??i,y;t[15]===v?y=t[16]:(y=v.toUpperCase(),t[15]=v,t[16]=y);let b;t[17]===Symbol.for(`react.memo_cache_sentinel`)?(b=()=>{l(null)},t[17]=b):b=t[17];let x;t[18]===a?x=t[19]:(x=e=>{let t=sa(e.target.value),n=ca(t);if(n==null){l(t);return}l(null),a(n)},t[18]=a,t[19]=x);let S;t[20]!==n||t[21]!==r||t[22]!==y||t[23]!==x?(S=(0,Q.jsx)(`input`,{"aria-label":n,className:`min-w-0 flex-1 bg-transparent text-[11px] uppercase tabular-nums outline-hidden disabled:cursor-default`,disabled:r,spellCheck:!1,type:`text`,value:y,onClick:ta,onBlur:b,onChange:x,onPointerDown:ea}),t[20]=n,t[21]=r,t[22]=y,t[23]=x,t[24]=S):S=t[24];let C;t[25]!==S||t[26]!==f||t[27]!==_?(C=(0,Q.jsxs)(`div`,{className:`relative flex h-7 w-full max-w-[8.5rem] items-center gap-2 rounded-lg border border-transparent px-2 shadow-sm max-sm:max-w-none`,style:f,children:[_,S]}),t[25]=S,t[26]=f,t[27]=_,t[28]=C):C=t[28];let w;t[29]===a?w=t[30]:(w=e=>{a(e)},t[29]=a,t[30]=w);let T;t[31]!==w||t[32]!==i?(T=(0,Q.jsx)(xt,{align:`end`,className:`w-auto rounded-xl p-3`,sideOffset:8,onOpenAutoFocus:$i,children:(0,Q.jsx)(wi,{className:`h-34 w-34`,color:i,onChange:w})}),t[31]=w,t[32]=i,t[33]=T):T=t[33];let E;return t[34]!==o||t[35]!==C||t[36]!==T?(E=(0,Q.jsxs)(St,{open:o,onOpenChange:u,children:[C,T]}),t[34]=o,t[35]=C,t[36]=T,t[37]=E):E=t[37],E}function $i(e){e.preventDefault()}function ea(e){e.stopPropagation()}function ta(e){e.stopPropagation()}function na(e){let t=(0,Y.c)(11),{ariaLabel:n,disabled:r,placeholder:i,value:a,onChange:o}=e,s=`${n}:${a??``}`,c=a??``,l,u;t[0]===o?(l=t[1],u=t[2]):(l=e=>{let t=e.currentTarget.value.trim();e.currentTarget.value=t,o(t.length>0?t:null)},u=e=>{if(e.key!==`Enter`)return;e.preventDefault();let t=e.currentTarget.value.trim();e.currentTarget.value=t,o(t.length>0?t:null)},t[0]=o,t[1]=l,t[2]=u);let d;return t[3]!==n||t[4]!==r||t[5]!==i||t[6]!==s||t[7]!==c||t[8]!==l||t[9]!==u?(d=(0,Q.jsx)(`input`,{"aria-label":n,className:`focus-visible:ring-token-focus h-7 w-full max-w-[8.5rem] rounded-lg border border-token-border bg-token-input-background px-2 text-[11px] text-token-text-primary shadow-sm outline-none focus-visible:ring-2 max-sm:max-w-none`,defaultValue:c,disabled:r,placeholder:i,spellCheck:!1,type:`text`,onBlur:l,onKeyDown:u},s),t[3]=n,t[4]=r,t[5]=i,t[6]=s,t[7]=c,t[8]=l,t[9]=u,t[10]=d):d=t[10],d}function ra(e){let t=(0,Y.c)(15),{ariaLabel:n,disabled:r,theme:i,value:a,onChange:o}=e,s;t[0]===o?s=t[1]:(s=e=>{o(Number(e.target.value))},t[0]=o,t[1]=s);let c=`linear-gradient(90deg, color-mix(in srgb, ${i.accent} 35%, ${i.surface}) 0%, ${i.accent} 32%, ${i.accent} 100%)`,l;t[2]===c?l=t[3]:(l={background:c,color:`var(--color-token-foreground)`},t[2]=c,t[3]=l);let u;t[4]!==n||t[5]!==r||t[6]!==s||t[7]!==l||t[8]!==a?(u=(0,Q.jsx)(`input`,{"aria-label":n,className:`h-0.5 flex-1 appearance-none rounded-full [&::-moz-range-thumb]:h-5 [&::-moz-range-thumb]:w-5 [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:border [&::-moz-range-thumb]:border-transparent [&::-moz-range-thumb]:bg-current [&::-moz-range-thumb]:shadow-sm [&::-moz-range-track]:h-0.5 [&::-moz-range-track]:rounded-full [&::-webkit-slider-runnable-track]:h-0.5 [&::-webkit-slider-runnable-track]:rounded-full [&::-webkit-slider-thumb]:mt-[-9px] [&::-webkit-slider-thumb]:h-5 [&::-webkit-slider-thumb]:w-5 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:border [&::-webkit-slider-thumb]:border-transparent [&::-webkit-slider-thumb]:bg-current [&::-webkit-slider-thumb]:shadow-sm`,disabled:r,max:100,min:0,onChange:s,step:1,style:l,type:`range`,value:a}),t[4]=n,t[5]=r,t[6]=s,t[7]=l,t[8]=a,t[9]=u):u=t[9];let d;t[10]===a?d=t[11]:(d=(0,Q.jsx)(`span`,{className:`w-9 text-right text-sm text-token-text-primary tabular-nums`,children:a}),t[10]=a,t[11]=d);let f;return t[12]!==u||t[13]!==d?(f=(0,Q.jsxs)(`div`,{className:`flex h-9 min-w-[12rem] items-center gap-2.5 max-sm:w-full max-sm:min-w-0`,children:[u,d]}),t[12]=u,t[13]=d,t[14]=f):f=t[14],f}function ia(e){return e===`light`?(0,Q.jsx)(I,{id:`settings.general.appearance.lightChromeTheme`,defaultMessage:`Light theme`,description:`Label for light chrome theme controls in appearance settings`}):(0,Q.jsx)(I,{id:`settings.general.appearance.darkChromeTheme`,defaultMessage:`Dark theme`,description:`Label for dark chrome theme controls in appearance settings`})}function aa(e,t){return t===`light`?e.formatMessage({id:`settings.general.appearance.theme.light`,defaultMessage:`Light`,description:`Light theme option`}):e.formatMessage({id:`settings.general.appearance.theme.dark`,defaultMessage:`Dark`,description:`Dark theme option`})}function oa(e){let t=la(e);return t==null||(t.red*.2126+t.green*.7152+t.blue*.0722)/255>.62?`#101010`:`#ffffff`}function sa(e){let t=e.toUpperCase().replace(/[^0-9A-F#]/g,``).replaceAll(`#`,``);return t.length===0?`#`:`#${t.slice(0,6)}`}function ca(e){return/^#[0-9A-F]{6}$/.test(e)?e.toLowerCase():null}function la(e){return/^#[0-9a-fA-F]{6}$/.test(e)?{blue:Number.parseInt(e.slice(5,7),16),green:Number.parseInt(e.slice(3,5),16),red:Number.parseInt(e.slice(1,3),16)}:null}var ua=new Set([`Meta`,`Control`,`Alt`,`AltGraph`,`Shift`]);function da(e){return ua.has(e)}function fa(e){return e==null?null:/^Key[A-Z]$/.test(e)?e.slice(3):/^Digit[0-9]$/.test(e)?e.slice(5):e===`Space`?`Space`:null}function pa(e,t){if(da(e))return null;let n=fa(t);if(n!=null)return n;if(e===` `||e===`\xA0`)return`Space`;if(e===`+`)return`Plus`;switch(e){case`Escape`:return`Esc`;case`ArrowUp`:return`Up`;case`ArrowDown`:return`Down`;case`ArrowLeft`:return`Left`;case`ArrowRight`:return`Right`;default:break}return/^f\d{1,2}$/i.test(e)||e.length===1?e.toUpperCase():e}function ma(e){let t=pa(e.key,e.code);if(t==null)return null;let n=[];return e.ctrlKey&&n.push(`Ctrl`),e.metaKey&&n.push(`Command`),e.altKey&&n.push(`Alt`),e.shiftKey&&n.push(`Shift`),n.push(t),n.join(`+`)}var ha=`STEPS_PROSE`,ga=`STEPS_COMMANDS`;function _a(){let{data:e}=K(h.CONVERSATION_DETAIL_MODE);return e??`STEPS_COMMANDS`}function va(){let e=(0,Y.c)(66),t=F(),{data:n,setData:r,isLoading:i}=te(O.NOTIFICATIONS_TURN_MODE),{data:a,setData:o,isLoading:s}=te(O.NOTIFICATIONS_PERMISSIONS_ENABLED),{data:c,setData:l,isLoading:u}=te(O.NOTIFICATIONS_QUESTIONS_ENABLED),d=n??`unfocused`,f;e[0]===Symbol.for(`react.memo_cache_sentinel`)?(f=(0,Q.jsx)(I,{id:`notifications.turnMode.off`,defaultMessage:`Never`,description:`Turn notification mode: never show notifications`}),e[0]=f):f=e[0];let p;e[1]===t?p=e[2]:(p=t.formatMessage({id:`notifications.turnMode.off`,defaultMessage:`Never`,description:`Turn notification mode: never show notifications`}),e[1]=t,e[2]=p);let m;e[3]===p?m=e[4]:(m={id:`off`,label:f,ariaLabel:p},e[3]=p,e[4]=m);let h;e[5]===Symbol.for(`react.memo_cache_sentinel`)?(h=(0,Q.jsx)(I,{id:`notifications.turnMode.unfocused`,defaultMessage:`Only when unfocused`,description:`Turn notification mode: only when app not focused`}),e[5]=h):h=e[5];let g;e[6]===t?g=e[7]:(g=t.formatMessage({id:`notifications.turnMode.unfocused`,defaultMessage:`Only when unfocused`,description:`Turn notification mode: only when app not focused`}),e[6]=t,e[7]=g);let _;e[8]===g?_=e[9]:(_={id:`unfocused`,label:h,ariaLabel:g},e[8]=g,e[9]=_);let v;e[10]===Symbol.for(`react.memo_cache_sentinel`)?(v=(0,Q.jsx)(I,{id:`notifications.turnMode.always`,defaultMessage:`Always`,description:`Turn notification mode: always show notifications`}),e[10]=v):v=e[10];let y;e[11]===t?y=e[12]:(y=t.formatMessage({id:`notifications.turnMode.always`,defaultMessage:`Always`,description:`Turn notification mode: always show notifications`}),e[11]=t,e[12]=y);let b;e[13]===y?b=e[14]:(b={id:`always`,label:v,ariaLabel:y},e[13]=y,e[14]=b);let x;e[15]!==m||e[16]!==_||e[17]!==b?(x=[m,_,b],e[15]=m,e[16]=_,e[17]=b,e[18]=x):x=e[18];let S=x,C;e[19]!==r||e[20]!==i?(C=e=>{i||r(e)},e[19]=r,e[20]=i,e[21]=C):C=e[21];let w=C,T;e[22]!==s||e[23]!==o?(T=e=>{s||o(!!e)},e[22]=s,e[23]=o,e[24]=T):T=e[24];let E=T,D;e[25]!==u||e[26]!==l?(D=e=>{u||l(!!e)},e[25]=u,e[26]=l,e[27]=D):D=e[27];let k=D,A,j;e[28]===Symbol.for(`react.memo_cache_sentinel`)?(A=(0,Q.jsx)(I,{id:`notifications.turnMode.label`,defaultMessage:`Turn completion notifications`,description:`Heading for turn completion notification settings`}),j=(0,Q.jsx)(I,{id:`notifications.turnMode.description`,defaultMessage:`Set when Codex alerts you that it's finished`,description:`Description for turn completion notification settings`}),e[28]=A,e[29]=j):(A=e[28],j=e[29]);let M=S.find(e=>e.id===d)?.label,N;e[30]===M?N=e[31]:(N=(0,Q.jsx)(`span`,{className:`truncate`,children:M}),e[30]=M,e[31]=N);let P;e[32]!==N||e[33]!==i?(P=(0,Q.jsx)(wt,{disabled:i,children:N}),e[32]=N,e[33]=i,e[34]=P):P=e[34];let L;e[35]!==w||e[36]!==d||e[37]!==i||e[38]!==S?(L=(0,Q.jsx)(`div`,{className:`max-h-80 overflow-y-auto`,children:S.map(e=>{let t=e.id===d;return(0,Q.jsx)(U.Item,{disabled:i,RightIcon:t?H:void 0,onSelect:()=>w(e.id),"aria-label":e.ariaLabel,children:(0,Q.jsx)(`span`,{className:`truncate`,children:e.label})},e.id)})}),e[35]=w,e[36]=d,e[37]=i,e[38]=S,e[39]=L):L=e[39];let R;e[40]!==P||e[41]!==L||e[42]!==i?(R=(0,Q.jsx)(q,{label:A,description:j,control:(0,Q.jsx)(he,{contentWidth:`menuWide`,disabled:i,align:`end`,triggerButton:P,children:L})}),e[40]=P,e[41]=L,e[42]=i,e[43]=R):R=e[43];let z,ee;e[44]===Symbol.for(`react.memo_cache_sentinel`)?(z=(0,Q.jsx)(I,{id:`notifications.permissions.label`,defaultMessage:`Enable permission notifications`,description:`Toggle label for permission notifications`}),ee=(0,Q.jsx)(I,{id:`notifications.permissions.description`,defaultMessage:`Show alerts when notification permissions are required`,description:`Description for permission notification toggle`}),e[44]=z,e[45]=ee):(z=e[44],ee=e[45]);let ne=a??!0,B;e[46]===t?B=e[47]:(B=t.formatMessage({id:`notifications.permissions.label`,defaultMessage:`Enable permission notifications`,description:`Toggle label for permission notifications`}),e[46]=t,e[47]=B);let re;e[48]!==E||e[49]!==s||e[50]!==ne||e[51]!==B?(re=(0,Q.jsx)(q,{label:z,description:ee,control:(0,Q.jsx)(ut,{checked:ne,disabled:s,onCheckedChange:E,"aria-label":B})}),e[48]=E,e[49]=s,e[50]=ne,e[51]=B,e[52]=re):re=e[52];let ie,ae;e[53]===Symbol.for(`react.memo_cache_sentinel`)?(ie=(0,Q.jsx)(I,{id:`notifications.questions.label`,defaultMessage:`Enable question notifications`,description:`Toggle label for question notifications`}),ae=(0,Q.jsx)(I,{id:`notifications.questions.description`,defaultMessage:`Show alerts when input is needed to continue`,description:`Description for question notification toggle`}),e[53]=ie,e[54]=ae):(ie=e[53],ae=e[54]);let oe=c??!0,se;e[55]===t?se=e[56]:(se=t.formatMessage({id:`notifications.questions.label`,defaultMessage:`Enable question notifications`,description:`Toggle label for question notifications`}),e[55]=t,e[56]=se);let V;e[57]!==k||e[58]!==u||e[59]!==oe||e[60]!==se?(V=(0,Q.jsx)(q,{label:ie,description:ae,control:(0,Q.jsx)(ut,{checked:oe,disabled:u,onCheckedChange:k,"aria-label":se})}),e[57]=k,e[58]=u,e[59]=oe,e[60]=se,e[61]=V):V=e[61];let ce;return e[62]!==R||e[63]!==re||e[64]!==V?(ce=(0,Q.jsxs)(Q.Fragment,{children:[R,re,V]}),e[62]=R,e[63]=re,e[64]=V,e[65]=ce):ce=e[65],ce}var ya=N({description:{id:`settings.agent.speed.description`,defaultMessage:`Choose how quickly inference runs across threads, subagents, and compaction. Fast uses 2x plan usage.`,description:`Description for the Fast mode speed setting`},label:{id:`settings.agent.speed.label`,defaultMessage:`Speed`,description:`Label for the Fast mode speed setting`},optionFast:{id:`settings.agent.speed.option.fast`,defaultMessage:`Fast`,description:`Label for the fast Speed setting option`},optionFastDescription:{id:`settings.agent.speed.option.fast.description`,defaultMessage:`1.5x speed, 2x plan usage`,description:`Subtitle for the fast Speed setting option`},optionStandard:{id:`settings.agent.speed.option.standard`,defaultMessage:`Standard`,description:`Label for the standard Speed setting option`},optionStandardDescription:{id:`settings.agent.speed.option.standard.description`,defaultMessage:`Default speed`,description:`Subtitle for the standard Speed setting option`}}),ba=Ur.map(e=>({label:Sa(e),description:Ca(e),value:e}));function xa(){let e=(0,Y.c)(35),t=F(),n=kt(),{serviceTierSettings:r,setServiceTier:i}=At();if(!n)return null;let a;e[0]===r.serviceTier?a=e[1]:(a=k(r.serviceTier),e[0]=r.serviceTier,e[1]=a);let o=a,s,c,l,u,d,f,p,m,h;if(e[2]!==t||e[3]!==o||e[4]!==r.isLoading){let n=ba.find(e=>e.value===o)??ba[0];l=q,e[14]===Symbol.for(`react.memo_cache_sentinel`)?(m=(0,Q.jsx)(I,{...ya.label}),h=(0,Q.jsx)(I,{...ya.description}),e[14]=m,e[15]=h):(m=e[14],h=e[15]),c=he,f=`menuWide`,p=`end`,s=wt,u=r.isLoading,d=t.formatMessage(n.label),e[2]=t,e[3]=o,e[4]=r.isLoading,e[5]=s,e[6]=c,e[7]=l,e[8]=u,e[9]=d,e[10]=f,e[11]=p,e[12]=m,e[13]=h}else s=e[5],c=e[6],l=e[7],u=e[8],d=e[9],f=e[10],p=e[11],m=e[12],h=e[13];let g;e[16]!==s||e[17]!==u||e[18]!==d?(g=(0,Q.jsx)(s,{disabled:u,children:d}),e[16]=s,e[17]=u,e[18]=d,e[19]=g):g=e[19];let _;e[20]!==o||e[21]!==r.isLoading||e[22]!==i?(_=ba.map(e=>{let t=e.value===o;return(0,Q.jsx)(U.Item,{disabled:r.isLoading,RightIcon:t?H:void 0,SubText:(0,Q.jsx)(`span`,{className:`text-token-description-foreground`,children:(0,Q.jsx)(I,{...e.description})}),onSelect:()=>{i(e.value,`settings`)},children:(0,Q.jsx)(I,{...e.label})},e.label.id)}),e[20]=o,e[21]=r.isLoading,e[22]=i,e[23]=_):_=e[23];let v;e[24]!==c||e[25]!==f||e[26]!==p||e[27]!==g||e[28]!==_?(v=(0,Q.jsx)(c,{contentWidth:f,align:p,triggerButton:g,children:_}),e[24]=c,e[25]=f,e[26]=p,e[27]=g,e[28]=_,e[29]=v):v=e[29];let y;return e[30]!==l||e[31]!==m||e[32]!==h||e[33]!==v?(y=(0,Q.jsx)(l,{label:m,description:h,control:v}),e[30]=l,e[31]=m,e[32]=h,e[33]=v,e[34]=y):y=e[34],y}function Sa(e){switch(e){case null:return ya.optionStandard;case`fast`:return ya.optionFast}}function Ca(e){switch(e){case null:return ya.optionStandardDescription;case`fast`:return ya.optionFastDescription}}var wa=`ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace`,Ta=[`powershell`,`commandPrompt`,`gitBash`,`wsl`],Ea;function Da(){let e=(0,Y.c)(27),t;e[0]===Symbol.for(`react.memo_cache_sentinel`)?(t=`2106641128`,e[0]=t):t=e[0];let n=B(t),r;e[1]===Symbol.for(`react.memo_cache_sentinel`)?(r=`1372061905`,e[1]=r):r=e[1];let i=B(r),a;e[2]===Symbol.for(`react.memo_cache_sentinel`)?(a=(0,Q.jsx)(Ct,{slug:`general-settings`}),e[2]=a):a=e[2];let o,s,c,l;e[3]===Symbol.for(`react.memo_cache_sentinel`)?(o=(0,Q.jsx)(V,{electron:!0,children:(0,Q.jsx)(Fa,{})}),s=(0,Q.jsx)(za,{}),c=(0,Q.jsx)(Ra,{}),l=(0,Q.jsx)(Wa,{}),e[3]=o,e[4]=s,e[5]=c,e[6]=l):(o=e[3],s=e[4],c=e[5],l=e[6]);let u;e[7]===Symbol.for(`react.memo_cache_sentinel`)?(u=(0,Q.jsx)(V,{electron:!0,children:(0,Q.jsx)(Va,{})}),e[7]=u):u=e[7];let d;e[8]===i?d=e[9]:(d=i?(0,Q.jsx)(Na,{}):null,e[8]=i,e[9]=d);let f;e[10]===d?f=e[11]:(f=(0,Q.jsx)(V,{electron:!0,children:d}),e[10]=d,e[11]=f);let p,m,h,g;e[12]===Symbol.for(`react.memo_cache_sentinel`)?(p=(0,Q.jsx)(V,{electron:!0,children:(0,Q.jsx)(Qa,{})}),m=(0,Q.jsx)(Za,{}),h=(0,Q.jsx)(xa,{}),g=(0,Q.jsx)(qa,{}),e[12]=p,e[13]=m,e[14]=h,e[15]=g):(p=e[12],m=e[13],h=e[14],g=e[15]);let _;e[16]===f?_=e[17]:(_=(0,Q.jsx)(J,{children:(0,Q.jsx)(J.Content,{children:(0,Q.jsxs)(Tt,{children:[o,s,c,l,u,f,p,m,h,g]})})}),e[16]=f,e[17]=_);let v;e[18]===Symbol.for(`react.memo_cache_sentinel`)?(v=(0,Q.jsx)(J.Header,{title:(0,Q.jsx)(I,{id:`settings.general.notifications`,defaultMessage:`Notifications`,description:`Heading for notifications settings group`})}),e[18]=v):v=e[18];let y;e[19]===Symbol.for(`react.memo_cache_sentinel`)?(y=(0,Q.jsx)(V,{electron:!0,children:(0,Q.jsxs)(J,{children:[v,(0,Q.jsx)(J.Content,{children:(0,Q.jsx)(Tt,{children:(0,Q.jsx)(va,{})})})]})}),e[19]=y):y=e[19];let b;e[20]===n?b=e[21]:(b=n?(0,Q.jsx)(ka,{}):null,e[20]=n,e[21]=b);let x;e[22]===b?x=e[23]:(x=(0,Q.jsx)(V,{electron:!0,children:b}),e[22]=b,e[23]=x);let S;return e[24]!==_||e[25]!==x?(S=(0,Q.jsxs)(L,{title:a,children:[_,y,x]}),e[24]=_,e[25]=x,e[26]=S):S=e[26],S}function Oa(){let e=(0,Y.c)(2),t;e[0]===Symbol.for(`react.memo_cache_sentinel`)?(t=(0,Q.jsx)(Ga,{}),e[0]=t):t=e[0];let n;return e[1]===Symbol.for(`react.memo_cache_sentinel`)?(n=(0,Q.jsx)(J,{children:(0,Q.jsx)(J.Content,{children:(0,Q.jsxs)(Tt,{children:[(0,Q.jsxs)(V,{electron:!0,children:[t,(0,Q.jsxs)(`div`,{className:`flex flex-col gap-2 p-1`,children:[(0,Q.jsx)(Gi,{}),(0,Q.jsx)(Ui,{})]}),(0,Q.jsx)(Ka,{})]}),(0,Q.jsx)(Ya,{}),(0,Q.jsx)(Xa,{})]})})}),e[1]=n):n=e[1],n}function ka(){let e=(0,Y.c)(22),t=F(),[n,r]=(0,Z.useState)(!1),i=B(zr),a=Vr(),o=pe(),{data:s,isLoading:c}=de(),l=s===void 0?[]:s,u=fe(),d=ue(),f=l.filter(Ma),p=i&&l.some(ja),m=t.formatMessage({id:`settings.general.experimentalFeatures.threadRealtime.name`,defaultMessage:`Realtime voice`,description:`Name of the experimental feature that enables realtime voice mode in thread composers`}),h;e[0]===t?h=e[1]:(h=t.formatMessage({id:`settings.general.experimentalFeatures.threadRealtime.description`,defaultMessage:`Talk to Codex in real time from the thread composer. Restart Codex after changing this setting.`,description:`Description of the experimental feature that enables realtime voice mode in thread composers`}),e[0]=t,e[1]=h);let g=h,_=l.find(Aa),v=_?.enabled??!1,y=[...o?[{key:`plugins`,label:t.formatMessage({id:`settings.general.experimentalFeatures.plugins.label`,defaultMessage:`Plugins`,description:`Label for the plugins experimental feature toggle`}),description:_?.description??t.formatMessage({id:`settings.general.experimentalFeatures.plugins.description`,defaultMessage:`Enable the plugins experience in Codex.`,description:`Description for the plugins experimental feature toggle`}),enabled:v,onChange:e=>{d.mutate({enabled:e},{onSuccess:()=>{r(!0)}})}}]:[],...f.map(e=>({key:e.name,label:e.displayName??e.name,description:e.description??void 0,enabled:e.enabled,onChange:t=>{u.mutate({featureName:e.name,enabled:t},{onSuccess:()=>{r(!0)}})}}))],b=u.isPending||d.isPending,x=y.length>0||p,S=J,C;e[2]===Symbol.for(`react.memo_cache_sentinel`)?(C=(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures`,defaultMessage:`Experimental features (Beta)`,description:`Heading for beta experimental features settings group`}),e[2]=C):C=e[2];let w;e[3]===n?w=e[4]:(w=n?(0,Q.jsx)(`div`,{className:`mb-2 block font-medium text-token-error-foreground`,children:(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.restartNote`,defaultMessage:`Restart Codex to apply experimental feature changes`,description:`Notice shown after changing an experimental feature to indicate restart is required`})}):void 0,e[3]=n,e[4]=w);let T;e[5]===w?T=e[6]:(T=(0,Q.jsx)(J.Header,{title:C,subtitle:w}),e[5]=w,e[6]=T);let E=J,D=Tt,O;e[7]===c?O=e[8]:(O=c?(0,Q.jsx)(q,{label:(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.loading`,defaultMessage:`Loading experimental features…`,description:`Loading label for beta experimental features settings group`}),control:(0,Q.jsx)(`span`,{className:`h-5 w-8`})}):null,e[7]=c,e[8]=O);let k=!c&&!x?(0,Q.jsx)(q,{label:(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.empty`,defaultMessage:`No beta experimental features available.`,description:`Empty label for beta experimental features settings group`}),control:(0,Q.jsx)(`span`,{className:`h-5 w-8`})}):null,A=y.map(e=>(0,Q.jsx)(q,{label:e.label,description:e.description,control:(0,Q.jsx)(yt,{checked:e.enabled,disabled:b,onChange:e.onChange,ariaLabel:t.formatMessage({id:`settings.general.experimentalFeatures.toggle`,defaultMessage:`Toggle {featureName}`,description:`Aria label for toggling a beta experimental feature`},{featureName:e.label})})},e.key)),j=p?(0,Q.jsx)(q,{label:m,description:g,control:(0,Q.jsx)(yt,{checked:a,disabled:u.isPending,onChange:e=>{u.mutate({featureName:Rr,enabled:e},{onSuccess:()=>{r(!0)}})},ariaLabel:t.formatMessage({id:`settings.general.experimentalFeatures.toggle`,defaultMessage:`Toggle {featureName}`,description:`Aria label for toggling a beta experimental feature`},{featureName:m})})}):null,M;e[9]!==D||e[10]!==O||e[11]!==k||e[12]!==A||e[13]!==j?(M=(0,Q.jsxs)(D,{children:[O,k,A,j]}),e[9]=D,e[10]=O,e[11]=k,e[12]=A,e[13]=j,e[14]=M):M=e[14];let N;e[15]!==M||e[16]!==E.Content?(N=(0,Q.jsx)(E.Content,{children:M}),e[15]=M,e[16]=E.Content,e[17]=N):N=e[17];let P;return e[18]!==S||e[19]!==N||e[20]!==T?(P=(0,Q.jsxs)(S,{children:[T,N]}),e[18]=S,e[19]=N,e[20]=T,e[21]=P):P=e[21],P}function Aa(e){return e.name===`plugins`}function ja(e){return e.name===`realtime_conversation`&&e.stage!==`beta`}function Ma(e){return e.stage===`beta`&&e.name!==`multi_agent`&&e.name!==`plugins`&&e.name!==`plugin`}function Na(){let e=(0,Y.c)(27),t=F(),n=b(),r=dt(),i=z()===`electron`,[a,o]=(0,Z.useState)(!1),[s,c]=(0,Z.useState)(null),l;e[0]===i?l=e[1]:(l={queryConfig:{enabled:i}},e[0]=i,e[1]=l);let{data:u}=_(`hotkey-window-hotkey-state`,l),d;e[2]!==r||e[3]!==n?(d={onSuccess:e=>{let t=x(`hotkey-window-hotkey-state`);n.setQueryData(t,e.state),r(t)}},e[2]=r,e[3]=n,e[4]=d):d=e[4];let f=C(`hotkey-window-set-hotkey`,d),p;e[5]!==t||e[6]!==f?(p=async e=>{c(null);try{let t=await f.mutateAsync({hotkey:e});t.success||c(t.error)}catch(e){let n=e;c(n instanceof Error?n.message:t.formatMessage({id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.errorGeneric`,defaultMessage:`Failed to update Popout Window hotkey.`,description:`Fallback error shown when hotkey window hotkey update fails`}))}},e[5]=t,e[6]=f,e[7]=p):p=e[7];let m=p;if(!i||u?.supported===!1)return null;let h=u?.configuredHotkey??null,g;e[8]!==h||e[9]!==t?(g=h==null?t.formatMessage({id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.off`,defaultMessage:`Off`,description:`Status label when hotkey window hotkey is disabled`}):ce(h),e[8]=h,e[9]=t,e[10]=g):g=e[10];let v=g,y;e[11]===Symbol.for(`react.memo_cache_sentinel`)?(y=(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.label`,defaultMessage:`Popout Window hotkey`,description:`Label for hotkey window hotkey setting`}),e[11]=y):y=e[11];let S;e[12]===Symbol.for(`react.memo_cache_sentinel`)?(S=(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.description`,defaultMessage:`Set a global shortcut for Popout Window. Leave unset to keep it off.`,description:`Description for hotkey window hotkey setting`}),e[12]=S):S=e[12];let w;e[13]===s?w=e[14]:(w=s?(0,Q.jsx)(`span`,{className:`text-token-error-foreground`,children:s}):null,e[13]=s,e[14]=w);let T;e[15]===w?T=e[16]:(T=(0,Q.jsxs)(`div`,{className:`flex flex-col gap-1`,children:[S,w]}),e[15]=w,e[16]=T);let E;e[17]!==m||e[18]!==h||e[19]!==v||e[20]!==t||e[21]!==a||e[22]!==f?(E=a?(0,Q.jsxs)(`div`,{className:`flex items-center gap-2`,children:[(0,Q.jsx)(`input`,{autoFocus:!0,readOnly:!0,value:t.formatMessage({id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.capturePrompt`,defaultMessage:`Press shortcut`,description:`Prompt shown while capturing hotkey window hotkey input`}),onBlur:()=>{o(!1)},onKeyDown:e=>{if(e.repeat)return;if(e.preventDefault(),e.stopPropagation(),e.key===`Escape`){o(!1);return}let t=ma(e.nativeEvent);t!=null&&(o(!1),m(t))},"aria-label":t.formatMessage({id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.captureAriaLabel`,defaultMessage:`Popout Window hotkey capture`,description:`Aria label for hotkey window hotkey capture input`}),className:`h-9 w-36 rounded-md border border-token-input-border bg-token-input-background px-2 text-sm text-token-input-foreground transition-colors outline-none focus:border-token-focus-border`}),(0,Q.jsx)(R,{color:`ghost`,size:`toolbar`,onMouseDown:Pa,onClick:()=>{o(!1)},children:(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.cancel`,defaultMessage:`Cancel`,description:`Button label to cancel hotkey window hotkey capture`})})]}):(0,Q.jsxs)(`div`,{className:`flex items-center gap-2`,children:[(0,Q.jsx)(`span`,{className:`min-w-20 text-right text-sm text-token-text-secondary`,children:v}),(0,Q.jsx)(R,{color:`secondary`,size:`toolbar`,disabled:f.isPending,onClick:()=>{c(null),o(!0)},children:h==null?(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.set`,defaultMessage:`Set`,description:`Button label to set hotkey window hotkey`}):(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.change`,defaultMessage:`Change`,description:`Button label to change hotkey window hotkey`})}),h==null?null:(0,Q.jsx)(R,{color:`ghost`,size:`toolbar`,disabled:f.isPending,onClick:()=>{m(null)},children:(0,Q.jsx)(I,{id:`settings.general.experimentalFeatures.hotkeyWindowHotkey.clear`,defaultMessage:`Clear`,description:`Button label to clear hotkey window hotkey`})})]}),e[17]=m,e[18]=h,e[19]=v,e[20]=t,e[21]=a,e[22]=f,e[23]=E):E=e[23];let D;return e[24]!==T||e[25]!==E?(D=(0,Q.jsx)(q,{label:y,description:T,control:E}),e[24]=T,e[25]=E,e[26]=D):D=e[26],D}function Pa(e){e.preventDefault()}function Fa(){let e=(0,Y.c)(42),t=z()===`electron`,n=b(),r=dt(),i;e[0]===Symbol.for(`react.memo_cache_sentinel`)?(i={cwd:null},e[0]=i):i=e[0];let a;e[1]===t?a=e[2]:(a={params:i,queryConfig:{enabled:t,staleTime:y.ONE_MINUTE}},e[1]=t,e[2]=a);let{data:o}=_(`open-in-targets`,a),s;e[3]!==r||e[4]!==n?(s={onSuccess:(e,t)=>{let i=x(`open-in-targets`,{cwd:null});n.setQueryData(i,e=>e&&{...e,preferredTarget:t.target}),r(i)}},e[3]=r,e[4]=n,e[5]=s):s=e[5];let c=C(`set-preferred-app`,s),l,u,d,f,p,m,h,g,v,S;if(e[6]!==o?.preferredTarget||e[7]!==o?.targets||e[8]!==t||e[9]!==c){let n=o?.targets??[],r=o?.preferredTarget??n.find(La)?.id??null,i=n.filter(Ia),a=i.find(e=>e.id===r);u=q,e[20]===Symbol.for(`react.memo_cache_sentinel`)?(S=(0,Q.jsx)(I,{id:`settings.ide.defaultOpenTarget.label`,defaultMessage:`Default open destination`,description:`Label for default open-in target setting`}),d=(0,Q.jsx)(I,{id:`settings.ide.defaultOpenTarget.description`,defaultMessage:`Where files and folders open by default`,description:`Description for default open-in target setting`}),e[20]=d,e[21]=S):(d=e[20],S=e[21]),l=he,m=`menuFixed`;let s=!t||i.length===0,_=a?(0,Q.jsx)(Ua,{icon:a.icon,label:a.label}):(0,Q.jsx)(I,{id:`settings.ide.defaultOpenTarget.placeholder`,defaultMessage:`No targets found`,description:`Placeholder for default open-in target select`});e[22]!==s||e[23]!==_?(h=(0,Q.jsx)(wt,{disabled:s,children:_}),e[22]=s,e[23]=_,e[24]=h):h=e[24],g=!t||i.length===0,v=`end`,f=`max-h-80 overflow-y-auto`;let y;e[25]===c?y=e[26]:(y=e=>(0,Q.jsx)(U.Item,{onSelect:()=>{c.mutate({target:e.id})},children:(0,Q.jsx)(Ua,{icon:e.icon,label:e.label})},e.id),e[25]=c,e[26]=y),p=i.map(y),e[6]=o?.preferredTarget,e[7]=o?.targets,e[8]=t,e[9]=c,e[10]=l,e[11]=u,e[12]=d,e[13]=f,e[14]=p,e[15]=m,e[16]=h,e[17]=g,e[18]=v,e[19]=S}else l=e[10],u=e[11],d=e[12],f=e[13],p=e[14],m=e[15],h=e[16],g=e[17],v=e[18],S=e[19];let w;e[27]!==f||e[28]!==p?(w=(0,Q.jsx)(`div`,{className:f,children:p}),e[27]=f,e[28]=p,e[29]=w):w=e[29];let T;e[30]!==l||e[31]!==w||e[32]!==m||e[33]!==h||e[34]!==g||e[35]!==v?(T=(0,Q.jsx)(l,{contentWidth:m,triggerButton:h,disabled:g,align:v,children:w}),e[30]=l,e[31]=w,e[32]=m,e[33]=h,e[34]=g,e[35]=v,e[36]=T):T=e[36];let E;return e[37]!==u||e[38]!==d||e[39]!==T||e[40]!==S?(E=(0,Q.jsx)(u,{label:S,description:d,control:T}),e[37]=u,e[38]=d,e[39]=T,e[40]=S,e[41]=E):E=e[41],E}function Ia(e){return e.available!==!1}function La(e){return e.default}function Ra(){let e=(0,Y.c)(44),{platform:t}=le(),n=z()===`electron`&&t===`windows`,r;e[0]===n?r=e[1]:(r={queryConfig:{enabled:n,staleTime:y.ONE_MINUTE}},e[0]=n,e[1]=r);let{data:i,isLoading:a}=_(`terminal-shell-options`,r),o;e[2]===n?o=e[3]:(o={enabled:n},e[2]=n,e[3]=o);let{data:s,setData:c,isLoading:l}=K(h.INTEGRATED_TERMINAL_SHELL,o);if(!n)return null;let u;e[4]===i?.availableShells?u=e[5]:(u=i?.availableShells??[],e[4]=i?.availableShells,e[5]=u);let d=u,f,p,m,g,v,b,x,S,C;if(e[6]!==d||e[7]!==s||e[8]!==l||e[9]!==a||e[10]!==c){let t;e[20]===d?t=e[21]:(t=e=>e===`gitBash`||e===`wsl`?d.includes(e):!0,e[20]=d,e[21]=t);let n=Ta.filter(t),r=s??`powershell`,i=n.find(e=>e===r)??n[0],o=a||l||i==null;p=q,e[22]===Symbol.for(`react.memo_cache_sentinel`)?(S=(0,Q.jsx)(I,{id:`settings.openIn.integratedTerminalShell.label`,defaultMessage:`Integrated terminal shell`,description:`Label for integrated terminal shell setting`}),C=(0,Q.jsx)(I,{id:`settings.openIn.integratedTerminalShell.description`,defaultMessage:`Choose which shell opens in the integrated terminal.`,description:`Description for integrated terminal shell setting`}),e[22]=S,e[23]=C):(S=e[22],C=e[23]),f=he;let u=i?(0,Q.jsx)(Ba,{value:i}):(0,Q.jsx)(I,{id:`settings.openIn.integratedTerminalShell.unavailable`,defaultMessage:`No shells available`,description:`Placeholder shown when no integrated terminal shell options are available`});e[24]!==o||e[25]!==u?(v=(0,Q.jsx)(wt,{disabled:o,children:u}),e[24]=o,e[25]=u,e[26]=v):v=e[26],b=`end`,x=o,m=`w-[220px] max-w-xs`;let h;e[27]!==r||e[28]!==c?(h=e=>(0,Q.jsx)(U.Item,{onSelect:()=>{c(e)},RightIcon:r===e?H:void 0,children:(0,Q.jsx)(`span`,{className:`text-sm`,children:(0,Q.jsx)(Ba,{value:e})})},e),e[27]=r,e[28]=c,e[29]=h):h=e[29],g=n.map(h),e[6]=d,e[7]=s,e[8]=l,e[9]=a,e[10]=c,e[11]=f,e[12]=p,e[13]=m,e[14]=g,e[15]=v,e[16]=b,e[17]=x,e[18]=S,e[19]=C}else f=e[11],p=e[12],m=e[13],g=e[14],v=e[15],b=e[16],x=e[17],S=e[18],C=e[19];let w;e[30]!==m||e[31]!==g?(w=(0,Q.jsx)(`div`,{className:m,children:g}),e[30]=m,e[31]=g,e[32]=w):w=e[32];let T;e[33]!==f||e[34]!==w||e[35]!==v||e[36]!==b||e[37]!==x?(T=(0,Q.jsx)(f,{triggerButton:v,align:b,disabled:x,children:w}),e[33]=f,e[34]=w,e[35]=v,e[36]=b,e[37]=x,e[38]=T):T=e[38];let E;return e[39]!==p||e[40]!==T||e[41]!==S||e[42]!==C?(E=(0,Q.jsx)(p,{label:S,description:C,control:T}),e[39]=p,e[40]=T,e[41]=S,e[42]=C,e[43]=E):E=e[43],E}function za(){let{data:e}=lt(),t=e?.platform===`win32`&&e?.hasWsl&&e?.isVsCodeRunningInsideWsl===!1,{data:n}=Fr(),r=Ir(),{data:i,setData:a,isLoading:o}=K(h.RUN_CODEX_IN_WSL,{enabled:t});if(!t||i===void 0)return null;Ea??=i;let s=[{value:!1,label:(0,Q.jsx)(I,{id:`settings.agentEnvironment.windowsNative`,defaultMessage:`Windows native`,description:`Option label for running the agent natively on Windows`}),description:(0,Q.jsx)(I,{id:`settings.agentEnvironment.windowsNative.description`,defaultMessage:`Run the agent directly in Windows`,description:`Description for the Windows native agent environment option`})},{value:!0,label:(0,Q.jsx)(I,{id:`settings.agentEnvironment.wsl`,defaultMessage:`Windows Subsystem for Linux`,description:`Option label for running the agent inside WSL`}),description:(0,Q.jsx)(I,{id:`settings.agentEnvironment.wsl.description`,defaultMessage:`Run the agent inside WSL`,description:`Description for the WSL agent environment option`})}],c=s.find(e=>e.value===i)??s[0],l=s.find(e=>e.value===Ea)??c;return(0,Q.jsx)(q,{label:(0,Q.jsx)(I,{id:`settings.agentEnvironment.label`,defaultMessage:`Agent environment`,description:`Label for the agent environment setting`}),description:(0,Q.jsxs)(Q.Fragment,{children:[(0,Q.jsx)(I,{id:`settings.agentEnvironment.description`,defaultMessage:`Choose where the agent runs on Windows`,description:`Description for the agent environment setting`}),i===Ea?null:(0,Q.jsxs)(Q.Fragment,{children:[(0,Q.jsx)(`span`,{className:`block`}),(0,Q.jsx)(`span`,{className:`text-token-error-foreground`,children:(0,Q.jsx)(I,{id:`settings.agentEnvironment.restartNotice`,defaultMessage:`Restart Codex to apply this change. The agent is still running in {currentEnvironment}.`,description:`Notice shown when the selected agent environment differs from the current pre-restart environment`,values:{currentEnvironment:l.label}})})]})]}),control:(0,Q.jsx)(he,{triggerButton:(0,Q.jsx)(wt,{disabled:o||r.isPending,children:c.label}),disabled:o||r.isPending,align:`end`,children:(0,Q.jsx)(`div`,{className:`w-[320px] max-w-xs space-y-1`,children:s.map(e=>(0,Q.jsx)(U.Item,{onSelect:()=>{(async()=>{e.value&&n!=null&&await r.mutateAsync(null),await a(e.value)})()},RightIcon:i===e.value?H:void 0,children:(0,Q.jsxs)(`div`,{className:`flex flex-col items-start gap-0.5`,children:[(0,Q.jsx)(`span`,{className:`text-sm`,children:e.label}),(0,Q.jsx)(`span`,{className:`text-xs text-token-text-secondary`,children:e.description})]})},String(e.value)))})})})}function Ba(e){let t=(0,Y.c)(2),{value:n}=e,r=g[n],i;return t[0]===r?i=t[1]:(i=(0,Q.jsx)(Q.Fragment,{children:r}),t[0]=r,t[1]=i),i}function Va(){let e=(0,Y.c)(16),{data:t,setData:n,isLoading:r}=K(h.CONVERSATION_DETAIL_MODE),i=t??`STEPS_COMMANDS`,a;e[0]===Symbol.for(`react.memo_cache_sentinel`)?(a={value:ha,label:(0,Q.jsx)(I,{id:`settings.conversationDetail.steps`,defaultMessage:`Steps`,description:`Label for steps-only conversation detail setting`}),description:(0,Q.jsx)(I,{id:`settings.conversationDetail.steps.description`,defaultMessage:`Hide commands and outputs`,description:`Description for steps-only conversation detail setting`})},e[0]=a):a=e[0];let o;e[1]===Symbol.for(`react.memo_cache_sentinel`)?(o={value:`STEPS_COMMANDS`,label:(0,Q.jsx)(I,{id:`settings.conversationDetail.stepsWithCommands`,defaultMessage:`Steps with code commands`,description:`Label for steps with code commands conversation detail setting`}),description:(0,Q.jsx)(I,{id:`settings.conversationDetail.stepsWithCommands.description`,defaultMessage:`Show commands, collapse output`,description:`Description for steps with code commands conversation detail setting`})},e[1]=o):o=e[1];let s;e[2]===Symbol.for(`react.memo_cache_sentinel`)?(s=[a,o,{value:`STEPS_EXECUTION`,label:(0,Q.jsx)(I,{id:`settings.conversationDetail.stepsWithOutput`,defaultMessage:`Steps with code output`,description:`Label for steps with code output conversation detail setting`}),description:(0,Q.jsx)(I,{id:`settings.conversationDetail.stepsWithOutput.description`,defaultMessage:`Show commands and expand output`,description:`Description for steps with code output conversation detail setting`})}],e[2]=s):s=e[2];let c=s,l;e[3]===i?l=e[4]:(l=c.find(e=>e.value===i)??c.find(Ha)??c[0],e[3]=i,e[4]=l);let u=l,d,f;e[5]===Symbol.for(`react.memo_cache_sentinel`)?(d=(0,Q.jsx)(I,{id:`settings.threadDetail.label`,defaultMessage:`Thread detail`,description:`Label for thread detail level setting`}),f=(0,Q.jsx)(I,{id:`settings.threadDetail.description`,defaultMessage:`Choose how much command output to show in threads`,description:`Description for thread detail level setting`}),e[5]=d,e[6]=f):(d=e[5],f=e[6]);let p;e[7]!==r||e[8]!==u.label?(p=(0,Q.jsx)(wt,{disabled:r,children:u.label}),e[7]=r,e[8]=u.label,e[9]=p):p=e[9];let m;e[10]!==i||e[11]!==n?(m=(0,Q.jsx)(`div`,{className:`w-[260px] max-w-xs space-y-1`,children:c.map(e=>(0,Q.jsx)(U.Item,{onSelect:()=>{n(e.value)},RightIcon:i===e.value?H:void 0,children:(0,Q.jsxs)(`div`,{className:`flex flex-col items-start gap-0.5`,children:[(0,Q.jsx)(`span`,{className:`text-sm`,children:e.label}),(0,Q.jsx)(`span`,{className:`text-xs text-token-text-secondary`,children:e.description})]})},e.value))}),e[10]=i,e[11]=n,e[12]=m):m=e[12];let g;return e[13]!==p||e[14]!==m?(g=(0,Q.jsx)(q,{label:d,description:f,control:(0,Q.jsx)(he,{triggerButton:p,align:`end`,children:m})}),e[13]=p,e[14]=m,e[15]=g):g=e[15],g}function Ha(e){return e.value===ga}function Ua(e){let t=(0,Y.c)(8),{icon:n,label:r}=e,i;t[0]!==n||t[1]!==r?(i=n?(0,Q.jsx)(`img`,{alt:typeof r==`string`?r:``,src:n,className:`icon-sm`}):null,t[0]=n,t[1]=r,t[2]=i):i=t[2];let a;t[3]===r?a=t[4]:(a=(0,Q.jsx)(`span`,{className:`truncate`,children:r}),t[3]=r,t[4]=a);let o;return t[5]!==i||t[6]!==a?(o=(0,Q.jsxs)(`span`,{className:`flex items-center gap-1.5`,children:[i,a]}),t[5]=i,t[6]=a,t[7]=o):o=t[7],o}function Wa(){let e=F(),t=ie(`72216192`)?.get(`enable_i18n`,!0),[n,r]=(0,Z.useState)(``),{logEventWithStatsig:i}=re(),{data:a,setData:o,isLoading:s}=K(h.LOCALE_OVERRIDE,{enabled:t}),c=(0,Z.useMemo)(()=>[{code:ir,label:$a(ir,ir),localizedLabel:$a(`en`,e.locale)},...sr().map(t=>({code:t.locale,label:$a(t.locale,t.locale),localizedLabel:$a(t.locale,e.locale)}))].sort((e,t)=>e.label.localeCompare(t.label)),[e.locale]),l=cr(a),u=c.find(e=>ur(e.code,a))??null,d=(0,Z.useMemo)(()=>{let e=n.trim().toLowerCase();return e?c.filter(t=>t.label.toLowerCase().includes(e)||t.localizedLabel.toLowerCase().includes(e)):c},[n,c]);if(!t)return null;let f=e=>{i(`codex_i18n_language_selected`,{selection:e??`auto`,surface:`settings`})};return(0,Q.jsx)(q,{label:(0,Q.jsx)(I,{id:`settings.ide.language.label`,defaultMessage:`Language`,description:`Label for language setting`}),description:(0,Q.jsx)(I,{id:`settings.ide.language.description`,defaultMessage:`Language for the app UI`,description:`Description for language setting`}),control:(0,Q.jsxs)(he,{contentWidth:`menuWide`,disabled:c.length===0,align:`end`,triggerButton:(0,Q.jsx)(wt,{disabled:c.length===0,children:u?u.label:e.formatMessage({id:`settings.ide.language.auto`,defaultMessage:`Auto Detect`,description:`Fallback label for auto language detect`})}),children:[(0,Q.jsx)(`div`,{className:`pb-1`,children:(0,Q.jsx)(me,{value:n,onChange:e=>r(e.target.value),placeholder:e.formatMessage({id:`settings.ide.language.search`,defaultMessage:`Search languages`,description:`Search placeholder for language picker`})})}),(0,Q.jsx)(U.Item,{disabled:s,RightIcon:l==null?H:void 0,onSelect:()=>{(async()=>{try{await o(null),f(null)}catch{}})()},children:(0,Q.jsx)(I,{id:`settings.ide.language.autoOption`,defaultMessage:`Auto Detect`,description:`Auto detect language option`})}),(0,Q.jsx)(`div`,{className:`max-h-80 overflow-y-auto`,children:d.map(e=>{let t=ur(e.code,a);return(0,Q.jsx)(U.Item,{disabled:s,RightIcon:t?H:void 0,onSelect:()=>{(async()=>{try{let t=lr(e.code)?ir:e.code;await o(t),f(t)}catch{}})()},children:(0,Q.jsxs)(`span`,{className:`truncate`,children:[e.label,e.localizedLabel===e.label?``:` • ${e.localizedLabel}`]})},e.code)})})]})})}function Ga(){let e=(0,Y.c)(28),t=F(),{data:n,setData:r,isLoading:i}=K(h.APPEARANCE_THEME),a=n??`system`,o;e[0]!==i||e[1]!==r?(o=e=>{i||r(e)},e[0]=i,e[1]=r,e[2]=o):o=e[2];let s=o,c,l;e[3]===Symbol.for(`react.memo_cache_sentinel`)?(c=(0,Q.jsx)(I,{id:`settings.general.appearance.theme`,defaultMessage:`Theme`,description:`Label for theme selector in appearance settings`}),l=(0,Q.jsx)(I,{id:`settings.general.appearance.theme.description`,defaultMessage:`Use light, dark, or match your system`,description:`Description for theme selector in appearance settings`}),e[3]=c,e[4]=l):(c=e[3],l=e[4]);let u;e[5]===Symbol.for(`react.memo_cache_sentinel`)?(u=(0,Q.jsx)(Ja,{icon:(0,Q.jsx)(Gr,{className:`icon-sm`}),label:(0,Q.jsx)(I,{id:`settings.general.appearance.theme.light`,defaultMessage:`Light`,description:`Light theme option`})}),e[5]=u):u=e[5];let d;e[6]===t?d=e[7]:(d=t.formatMessage({id:`settings.general.appearance.theme.light`,defaultMessage:`Light`,description:`Light theme option`}),e[6]=t,e[7]=d);let f;e[8]===d?f=e[9]:(f={id:`light`,label:u,ariaLabel:d},e[8]=d,e[9]=f);let p;e[10]===Symbol.for(`react.memo_cache_sentinel`)?(p=(0,Q.jsx)(Ja,{icon:(0,Q.jsx)(Wr,{className:`icon-sm`}),label:(0,Q.jsx)(I,{id:`settings.general.appearance.theme.dark`,defaultMessage:`Dark`,description:`Dark theme option`})}),e[10]=p):p=e[10];let m;e[11]===t?m=e[12]:(m=t.formatMessage({id:`settings.general.appearance.theme.dark`,defaultMessage:`Dark`,description:`Dark theme option`}),e[11]=t,e[12]=m);let g;e[13]===m?g=e[14]:(g={id:`dark`,label:p,ariaLabel:m},e[13]=m,e[14]=g);let _;e[15]===Symbol.for(`react.memo_cache_sentinel`)?(_=(0,Q.jsx)(Ja,{icon:(0,Q.jsx)(mr,{className:`icon-sm`}),label:(0,Q.jsx)(I,{id:`settings.general.appearance.theme.system`,defaultMessage:`System`,description:`System theme option`})}),e[15]=_):_=e[15];let v;e[16]===t?v=e[17]:(v=t.formatMessage({id:`settings.general.appearance.theme.system`,defaultMessage:`System`,description:`System theme option`}),e[16]=t,e[17]=v);let y;e[18]===v?y=e[19]:(y={id:`system`,label:_,ariaLabel:v},e[18]=v,e[19]=y);let b;e[20]!==y||e[21]!==f||e[22]!==g?(b=[f,g,y],e[20]=y,e[21]=f,e[22]=g,e[23]=b):b=e[23];let x;return e[24]!==s||e[25]!==a||e[26]!==b?(x=(0,Q.jsx)(q,{label:c,description:l,control:(0,Q.jsx)(vt,{selectedId:a,onSelect:s,options:b})}),e[24]=s,e[25]=a,e[26]=b,e[27]=x):x=e[27],x}function Ka(){let e=(0,Y.c)(11),t=F(),{data:n,setData:r,isLoading:i}=K(h.USE_POINTER_CURSORS),a,o;e[0]===Symbol.for(`react.memo_cache_sentinel`)?(a=(0,Q.jsx)(I,{id:`settings.general.appearance.usePointerCursors.label`,defaultMessage:`Use pointer cursors`,description:`Label for pointer cursor interaction setting`}),o=(0,Q.jsx)(I,{id:`settings.general.appearance.usePointerCursors.description`,defaultMessage:`Change the cursor to a pointer when hovering over interactive elements`,description:`Description for pointer cursor interaction setting`}),e[0]=a,e[1]=o):(a=e[0],o=e[1]);let s=n===!0,c;e[2]===r?c=e[3]:(c=e=>{r(e)},e[2]=r,e[3]=c);let l;e[4]===t?l=e[5]:(l=t.formatMessage({id:`settings.general.appearance.usePointerCursors.label`,defaultMessage:`Use pointer cursors`,description:`Label for pointer cursor interaction setting`}),e[4]=t,e[5]=l);let u;return e[6]!==i||e[7]!==s||e[8]!==c||e[9]!==l?(u=(0,Q.jsx)(q,{label:a,description:o,control:(0,Q.jsx)(yt,{checked:s,disabled:i,onChange:c,ariaLabel:l})}),e[6]=i,e[7]=s,e[8]=c,e[9]=l,e[10]=u):u=e[10],u}function qa(){let e=(0,Y.c)(23),t=F(),{mode:n,setMode:r,isLoading:i}=pr(),a;e[0]===Symbol.for(`react.memo_cache_sentinel`)?(a=ce(`CmdOrCtrl+Shift+Enter`),e[0]=a):a=e[0];let o=a,s;e[1]!==i||e[2]!==r?(s=e=>{i||r(e)},e[1]=i,e[2]=r,e[3]=s):s=e[3];let c=s,l;e[4]===Symbol.for(`react.memo_cache_sentinel`)?(l=(0,Q.jsx)(I,{id:`settings.general.followUpQueueMode.label`,defaultMessage:`Follow-up behavior`,description:`Label for follow-up queue mode setting`}),e[4]=l):l=e[4];let u;e[5]===Symbol.for(`react.memo_cache_sentinel`)?(u=(0,Q.jsx)(I,{id:`settings.general.followUpQueueMode.description`,defaultMessage:`Queue follow-ups while Codex runs or steer the current run. Press {invertFollowUpShortcutLabel} to do the opposite for one message.`,description:`Description for follow-up queue mode setting`,values:{invertFollowUpShortcutLabel:o}}),e[5]=u):u=e[5];let d;e[6]===Symbol.for(`react.memo_cache_sentinel`)?(d=(0,Q.jsx)(I,{id:`settings.general.followUpQueueMode.queue`,defaultMessage:`Queue`,description:`Queue follow-up option label`}),e[6]=d):d=e[6];let f;e[7]===t?f=e[8]:(f=t.formatMessage({id:`settings.general.followUpQueueMode.queue`,defaultMessage:`Queue`,description:`Queue follow-up option label`}),e[7]=t,e[8]=f);let p;e[9]===f?p=e[10]:(p={id:`queue`,label:d,ariaLabel:f},e[9]=f,e[10]=p);let m;e[11]===Symbol.for(`react.memo_cache_sentinel`)?(m=(0,Q.jsx)(I,{id:`settings.general.followUpQueueMode.interrupt`,defaultMessage:`Steer`,description:`Steer follow-up option label`}),e[11]=m):m=e[11];let h;e[12]===t?h=e[13]:(h=t.formatMessage({id:`settings.general.followUpQueueMode.interrupt`,defaultMessage:`Steer`,description:`Steer follow-up option label`}),e[12]=t,e[13]=h);let g;e[14]===h?g=e[15]:(g={id:`steer`,label:m,ariaLabel:h},e[14]=h,e[15]=g);let _;e[16]!==p||e[17]!==g?(_=[p,g],e[16]=p,e[17]=g,e[18]=_):_=e[18];let v;return e[19]!==c||e[20]!==n||e[21]!==_?(v=(0,Q.jsx)(q,{className:`gap-6`,label:l,description:u,control:(0,Q.jsx)(vt,{selectedId:n,onSelect:c,options:_})}),e[19]=c,e[20]=n,e[21]=_,e[22]=v):v=e[22],v}function Ja(e){let t=(0,Y.c)(5),{icon:n,label:r}=e,i;t[0]===r?i=t[1]:(i=(0,Q.jsx)(`span`,{className:`text-sm`,children:r}),t[0]=r,t[1]=i);let a;return t[2]!==n||t[3]!==i?(a=(0,Q.jsxs)(`span`,{className:`flex items-center gap-1.5`,children:[n,i]}),t[2]=n,t[3]=i,t[4]=a):a=t[4],a}function Ya(){let e=(0,Y.c)(25),t=z()===`electron`,n=F(),r;e[0]===t?r=e[1]:(r={enabled:t},e[0]=t,e[1]=r);let{data:i,setData:a,isLoading:o}=K(h.SANS_FONT_SIZE,r),s=i??13,c=o,l;e[2]!==s||e[3]!==a?(l=e=>{let t=Number.parseFloat(e.value);if(Number.isNaN(t)){e.value=String(s);return}e.value=String(t),t!==s&&a(t)},e[2]=s,e[3]=a,e[4]=l):l=e[4];let u=l;if(!t)return null;let d,f;e[5]===Symbol.for(`react.memo_cache_sentinel`)?(d=(0,Q.jsx)(I,{id:`settings.general.appearance.sansFontSize.row`,defaultMessage:`UI font size`,description:`Label for UI font size setting`}),f=(0,Q.jsx)(I,{id:`settings.general.appearance.sansFontSize.row.description`,defaultMessage:`Adjust the base size used for the Codex UI`,description:`Description for UI font size setting`}),e[5]=d,e[6]=f):(d=e[5],f=e[6]);let p,m;e[7]===u?(p=e[8],m=e[9]):(p=e=>{u(e.currentTarget)},m=e=>{e.key===`Enter`&&(e.preventDefault(),u(e.currentTarget))},e[7]=u,e[8]=p,e[9]=m);let g;e[10]===n?g=e[11]:(g=n.formatMessage({id:`settings.general.appearance.sansFontSize`,defaultMessage:`Sans font size`,description:`Label for sans font size setting`}),e[10]=n,e[11]=g);let _;e[12]!==s||e[13]!==c||e[14]!==p||e[15]!==m||e[16]!==g?(_=(0,Q.jsx)(`input`,{className:`focus-visible:ring-token-focus h-token-button-composer w-16 rounded-lg border border-token-border bg-token-input-background px-2 py-0 text-right text-sm text-token-text-primary shadow-sm outline-none focus-visible:ring-2`,type:`number`,min:11,max:16,step:1,disabled:c,defaultValue:s,onBlur:p,onKeyDown:m,"aria-label":g},s),e[12]=s,e[13]=c,e[14]=p,e[15]=m,e[16]=g,e[17]=_):_=e[17];let v;e[18]===n?v=e[19]:(v=n.formatMessage({id:`settings.general.appearance.sansFontSize.units`,defaultMessage:`px`,description:`Unit label for sans font size setting`}),e[18]=n,e[19]=v);let y;e[20]===v?y=e[21]:(y=(0,Q.jsx)(`span`,{className:`text-sm text-token-text-secondary`,children:v}),e[20]=v,e[21]=y);let b;return e[22]!==_||e[23]!==y?(b=(0,Q.jsx)(q,{label:d,description:f,control:(0,Q.jsxs)(`div`,{className:`flex items-center gap-2`,children:[_,y]})}),e[22]=_,e[23]=y,e[24]=b):b=e[24],b}function Xa(){let e=(0,Y.c)(22),t=F(),{data:n,setData:r}=K(h.CODE_FONT_SIZE),i=n??12,a;e[0]!==i||e[1]!==r?(a=e=>{let t=Number.parseFloat(e.value);if(Number.isNaN(t)){e.value=String(i);return}e.value=String(t),t!==i&&r(t)},e[0]=i,e[1]=r,e[2]=a):a=e[2];let o=a,s,c;e[3]===Symbol.for(`react.memo_cache_sentinel`)?(s=(0,Q.jsx)(I,{id:`settings.general.appearance.codeFontSize.row`,defaultMessage:`Code font size`,description:`Label for code font size controls`}),c=(0,Q.jsx)(I,{id:`settings.general.appearance.codeFontSize.row.description`,defaultMessage:`Adjust the base size used for code across chats and diffs`,description:`Description for code font size controls`}),e[3]=s,e[4]=c):(s=e[3],c=e[4]);let l,u;e[5]===o?(l=e[6],u=e[7]):(l=e=>{o(e.currentTarget)},u=e=>{e.key===`Enter`&&(e.preventDefault(),o(e.currentTarget))},e[5]=o,e[6]=l,e[7]=u);let d;e[8]===t?d=e[9]:(d=t.formatMessage({id:`settings.general.appearance.codeFontSize`,defaultMessage:`Code font size`,description:`Label for code font size setting`}),e[8]=t,e[9]=d);let f;e[10]!==i||e[11]!==l||e[12]!==u||e[13]!==d?(f=(0,Q.jsx)(`input`,{className:`focus-visible:ring-token-focus h-token-button-composer w-16 rounded-lg border border-token-border bg-token-input-background px-2 py-0 text-right text-sm text-token-text-primary shadow-sm outline-none focus-visible:ring-2`,type:`number`,min:8,max:24,step:1,defaultValue:i,onBlur:l,onKeyDown:u,"aria-label":d},i),e[10]=i,e[11]=l,e[12]=u,e[13]=d,e[14]=f):f=e[14];let p;e[15]===t?p=e[16]:(p=t.formatMessage({id:`settings.general.appearance.codeFontSize.units`,defaultMessage:`px`,description:`Unit label for code font size setting`}),e[15]=t,e[16]=p);let m;e[17]===p?m=e[18]:(m=(0,Q.jsx)(`span`,{className:`text-sm text-token-text-secondary`,children:p}),e[17]=p,e[18]=m);let g;return e[19]!==f||e[20]!==m?(g=(0,Q.jsx)(q,{label:s,description:c,control:(0,Q.jsxs)(`div`,{className:`flex items-center gap-2`,children:[f,m]})}),e[19]=f,e[20]=m,e[21]=g):g=e[21],g}function Za(){let e=(0,Y.c)(14),{enterBehavior:t,setEnterBehavior:n,isLoading:r}=rr(),{modifierSymbol:i}=le(),a=t===`cmdIfMultiline`,o;e[0]===i?o=e[1]:(o=(0,Q.jsx)(I,{id:`settings.general.enterBehavior.label`,defaultMessage:`Require {modifierSymbol} + enter to send long prompts`,description:`Label for the enter key behavior toggle`,values:{modifierSymbol:i}}),e[0]=i,e[1]=o);let s;e[2]===i?s=e[3]:(s=(0,Q.jsx)(I,{id:`settings.general.enterBehavior.description`,defaultMessage:`When enabled, multiline prompts require {modifierSymbol} + enter to send.`,description:`Description for the enter key behavior toggle`,values:{modifierSymbol:i}}),e[2]=i,e[3]=s);let c;e[4]===n?c=e[5]:(c=e=>{n(e?`cmdIfMultiline`:`enter`)},e[4]=n,e[5]=c);let l;e[6]!==r||e[7]!==a||e[8]!==c?(l=(0,Q.jsx)(yt,{checked:a,disabled:r,onChange:c}),e[6]=r,e[7]=a,e[8]=c,e[9]=l):l=e[9];let u;return e[10]!==o||e[11]!==s||e[12]!==l?(u=(0,Q.jsx)(q,{label:o,description:s,control:l}),e[10]=o,e[11]=s,e[12]=l,e[13]=u):u=e[13],u}function Qa(){let e=(0,Y.c)(13),t=z(),{platform:n}=le(),r=t===`electron`&&n!==`windows`,i=F(),a;e[0]===r?a=e[1]:(a={enabled:r},e[0]=r,e[1]=a);let{data:o,setData:s,isLoading:c}=K(h.PREVENT_SLEEP_WHILE_RUNNING,a);if(!r)return null;let l,u;e[2]===Symbol.for(`react.memo_cache_sentinel`)?(l=(0,Q.jsx)(I,{id:`settings.general.power.preventSleepWhileRunning.label`,defaultMessage:`Prevent sleep while running`,description:`Label for preventing mac sleep while a thread runs`}),u=(0,Q.jsx)(I,{id:`settings.general.power.preventSleepWhileRunning.description`,defaultMessage:`Keep your computer awake while Codex is running a thread.`,description:`Description for preventing sleep while a thread runs`}),e[2]=l,e[3]=u):(l=e[2],u=e[3]);let d=o??!1,f;e[4]===s?f=e[5]:(f=e=>{s(e)},e[4]=s,e[5]=f);let p;e[6]===i?p=e[7]:(p=i.formatMessage({id:`settings.general.power.preventSleepWhileRunning.label`,defaultMessage:`Prevent sleep while running`,description:`Label for preventing mac sleep while a thread runs`}),e[6]=i,e[7]=p);let m;return e[8]!==c||e[9]!==d||e[10]!==f||e[11]!==p?(m=(0,Q.jsx)(q,{label:l,description:u,control:(0,Q.jsx)(yt,{checked:d,disabled:c,onChange:f,ariaLabel:p})}),e[8]=c,e[9]=d,e[10]=f,e[11]=p,e[12]=m):m=e[12],m}function $a(e,t){try{return new Intl.DisplayNames([t],{type:`language`,languageDisplay:`standard`}).of(e)??e}catch{return e}}export{rr as C,Dt as D,kt as E,cr as S,At as T,dr as _,ha as a,ur as b,Ur as c,Fr as d,Mr as f,ir as g,pr as h,Na as i,Vr as l,mr as m,wa as n,_a as o,Ar as p,Da as r,Gr as s,Oa as t,Ir as u,sr as v,Vn as w,fr as x,lr as y};
//# sourceMappingURL=general-settings-BcxoeEmb.js.map