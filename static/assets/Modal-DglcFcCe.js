import{A as e,D as t,E as n,K as r,L as i,N as a,P as o,S as s,Y as c,at as l,c as u,it as d,jt as f,k as p,l as m,lt as h,w as g,y as _}from"./runtime-core.esm-bundler-jDnCq53A.js";import{Dt as v,H as y,N as b,P as x,R as S,St as C,_t as w,dt as T,gt as E,lt as D,mt as O,n as k,pt as A,r as j,t as ee,ut as M,z as te}from"./light-Vq2WEi6e.js";import{d as N,l as ne,m as re,o as ie,p as P}from"./Loading-C7sDZO8k.js";import{c as F,l as ae,r as oe,s as I,t as se}from"./Scrollbar-DmQk0fnf.js";import{l as ce,n as le}from"./replaceable-BKB0h89t.js";import{a as ue,c as de,d as fe,f as pe,h as me,i as he,m as ge,n as L,o as _e,p as R,s as ve,t as ye}from"./fade-in-scale-up.cssr-DWlR83Lu.js";import{n as z,t as B}from"./utils-DqNdvgil.js";import{t as be}from"./Close-E-jUOvhm.js";import{t as xe}from"./is-browser-DrnB8jUw.js";import{t as Se}from"./event-DkY4jc81.js";import{t as V}from"./keysOf-7YXhwv04.js";import{t as H}from"./render-oEplYqTt.js";import{n as Ce,r as we,t as Te}from"./Success-dcJol_9t.js";import{t as Ee}from"./Warning--Xy0bbUk.js";import{t as De}from"./fade-in.cssr-Lj6AJNgN.js";import{i as Oe,t as ke}from"./Button-BdCOUxCk.js";import{a as Ae,n as je,r as Me,t as Ne}from"./Card-xrcOaho8.js";import{n as Pe}from"./context-JH0iwfQ7.js";var U=l(null);function Fe(e){if(e.clientX>0||e.clientY>0)U.value={x:e.clientX,y:e.clientY};else{let{target:t}=e;if(t instanceof Element){let{left:e,top:n,width:r,height:i}=t.getBoundingClientRect();e>0||n>0?U.value={x:e+r/2,y:n+i/2}:U.value={x:0,y:0}}else U.value=null}}var W=0,Ie=!0;function Le(){if(!z)return d(l(null));W===0&&F(`click`,document,Fe,!0);let t=()=>{W+=1};return(Ie&&=B())?(p(t),e(()=>{--W,W===0&&I(`click`,document,Fe,!0)})):t(),d(U)}var Re=l(void 0),G=0;function ze(){Re.value=Date.now()}var Be=!0;function Ve(t){if(!z)return d(l(!1));let n=l(!1),r=null;function i(){r!==null&&window.clearTimeout(r)}function a(){i(),n.value=!0,r=window.setTimeout(()=>{n.value=!1},t)}G===0&&F(`click`,window,ze,!0);let o=()=>{G+=1,F(`click`,window,a,!0)};return(Be&&=B())?(p(o),e(()=>{--G,G===0&&I(`click`,window,ze,!0),I(`click`,window,a,!0),i()})):o(),d(n)}var K=l(!1);function He(){K.value=!0}function Ue(){K.value=!1}var q=0;function We(){return xe&&(p(()=>{q||(window.addEventListener(`compositionstart`,He),window.addEventListener(`compositionend`,Ue)),q++}),e(()=>{q<=1?(window.removeEventListener(`compositionstart`,He),window.removeEventListener(`compositionend`,Ue),q=0):q--})),K}var J=0,Ge=``,Ke=``,qe=``,Je=``,Y=l(`0px`);function Ye(t){if(typeof document>`u`)return;let n=document.documentElement,i,o=!1,s=()=>{n.style.marginRight=Ge,n.style.overflow=Ke,n.style.overflowX=qe,n.style.overflowY=Je,Y.value=`0px`};a(()=>{i=r(t,e=>{if(e){if(!J){let e=window.innerWidth-n.offsetWidth;e>0&&(Ge=n.style.marginRight,n.style.marginRight=`${e}px`,Y.value=`${e}px`),Ke=n.style.overflow,qe=n.style.overflowX,Je=n.style.overflowY,n.style.overflow=`hidden`,n.style.overflowX=`hidden`,n.style.overflowY=`hidden`}o=!0,J++}else J--,J||s(),o=!1},{immediate:!0})}),e(()=>{i?.(),o&&=(J--,J||s(),!1)})}var Xe={titleFontSize:`18px`,padding:`16px 28px 20px 28px`,iconSize:`28px`,actionSpace:`12px`,contentMargin:`8px 0 16px 0`,iconMargin:`0 4px 0 0`,iconMarginIconTop:`4px 0 8px 0`,closeSize:`22px`,closeIconSize:`18px`,closeMargin:`20px 26px 0 0`,closeMarginIconTop:`10px 16px 0 0`};function Ze(e){let{textColor1:t,textColor2:n,modalColor:r,closeIconColor:i,closeIconColorHover:a,closeIconColorPressed:o,closeColorHover:s,closeColorPressed:c,infoColor:l,successColor:u,warningColor:d,errorColor:f,primaryColor:p,dividerColor:m,borderRadius:h,fontWeightStrong:g,lineHeight:_,fontSize:v}=e;return Object.assign(Object.assign({},Xe),{fontSize:v,lineHeight:_,border:`1px solid ${m}`,titleTextColor:t,textColor:n,color:r,closeColorHover:s,closeColorPressed:c,closeIconColor:i,closeIconColorHover:a,closeIconColorPressed:o,closeBorderRadius:h,iconColor:p,iconColorInfo:l,iconColorSuccess:u,iconColorWarning:d,iconColorError:f,borderRadius:h,titleFontWeight:g})}var Qe=k({name:`Dialog`,common:ee,peers:{Button:Oe},self:Ze}),X={icon:Function,type:{type:String,default:`default`},title:[String,Function],closable:{type:Boolean,default:!0},negativeText:String,positiveText:String,positiveButtonProps:Object,negativeButtonProps:Object,content:[String,Function],action:Function,showIcon:{type:Boolean,default:!0},loading:Boolean,bordered:Boolean,iconPlacement:String,titleClass:[String,Array],titleStyle:[String,Object],contentClass:[String,Array],contentStyle:[String,Object],actionClass:[String,Array],actionStyle:[String,Object],onPositiveClick:Function,onNegativeClick:Function,onClose:Function,closeFocusable:Boolean},$e=V(X),et=M([T(`dialog`,`
 --n-icon-margin: var(--n-icon-margin-top) var(--n-icon-margin-right) var(--n-icon-margin-bottom) var(--n-icon-margin-left);
 word-break: break-word;
 line-height: var(--n-line-height);
 position: relative;
 background: var(--n-color);
 color: var(--n-text-color);
 box-sizing: border-box;
 margin: auto;
 border-radius: var(--n-border-radius);
 padding: var(--n-padding);
 transition: 
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `,[A(`icon`,`
 color: var(--n-icon-color);
 `),O(`bordered`,`
 border: var(--n-border);
 `),O(`icon-top`,[A(`close`,`
 margin: var(--n-close-margin);
 `),A(`icon`,`
 margin: var(--n-icon-margin);
 `),A(`content`,`
 text-align: center;
 `),A(`title`,`
 justify-content: center;
 `),A(`action`,`
 justify-content: center;
 `)]),O(`icon-left`,[A(`icon`,`
 margin: var(--n-icon-margin);
 `),O(`closable`,[A(`title`,`
 padding-right: calc(var(--n-close-size) + 6px);
 `)])]),A(`close`,`
 position: absolute;
 right: 0;
 top: 0;
 margin: var(--n-close-margin);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 z-index: 1;
 `),A(`content`,`
 font-size: var(--n-font-size);
 margin: var(--n-content-margin);
 position: relative;
 word-break: break-word;
 `,[O(`last`,`margin-bottom: 0;`)]),A(`action`,`
 display: flex;
 justify-content: flex-end;
 `,[M(`> *:not(:last-child)`,`
 margin-right: var(--n-action-space);
 `)]),A(`icon`,`
 font-size: var(--n-icon-size);
 transition: color .3s var(--n-bezier);
 `),A(`title`,`
 transition: color .3s var(--n-bezier);
 display: flex;
 align-items: center;
 font-size: var(--n-title-font-size);
 font-weight: var(--n-title-font-weight);
 color: var(--n-title-text-color);
 `),T(`dialog-icon-container`,`
 display: flex;
 justify-content: center;
 `)]),w(T(`dialog`,`
 width: 446px;
 max-width: calc(100vw - 32px);
 `)),T(`dialog`,[D(`
 width: 446px;
 max-width: calc(100vw - 32px);
 `)])]),tt={default:()=>s(Ce,null),info:()=>s(Ce,null),success:()=>s(Te,null),warning:()=>s(Ee,null),error:()=>s(we,null)},nt=_({name:`Dialog`,alias:[`NimbusConfirmCard`,`Confirm`],props:Object.assign(Object.assign({},j.props),X),slots:Object,setup(e){let{mergedComponentPropsRef:t,mergedClsPrefixRef:n,inlineThemeDisabled:r,mergedRtlRef:i}=x(e),a=ie(`Dialog`,i,n),o=m(()=>{let{iconPlacement:n}=e;return n||t?.value?.Dialog?.iconPlacement||`left`});function s(t){let{onPositiveClick:n}=e;n&&n(t)}function c(t){let{onNegativeClick:n}=e;n&&n(t)}function l(){let{onClose:t}=e;t&&t()}let u=j(`Dialog`,`-dialog`,et,Qe,e,n),d=m(()=>{let{type:t}=e,n=o.value,{common:{cubicBezierEaseInOut:r},self:{fontSize:i,lineHeight:a,border:s,titleTextColor:c,textColor:l,color:d,closeBorderRadius:f,closeColorHover:p,closeColorPressed:m,closeIconColor:h,closeIconColorHover:g,closeIconColorPressed:_,closeIconSize:v,borderRadius:y,titleFontWeight:b,titleFontSize:x,padding:S,iconSize:C,actionSpace:w,contentMargin:T,closeSize:D,[n===`top`?`iconMarginIconTop`:`iconMargin`]:O,[n===`top`?`closeMarginIconTop`:`closeMargin`]:k,[E(`iconColor`,t)]:A}}=u.value,j=ce(O);return{"--n-font-size":i,"--n-icon-color":A,"--n-bezier":r,"--n-close-margin":k,"--n-icon-margin-top":j.top,"--n-icon-margin-right":j.right,"--n-icon-margin-bottom":j.bottom,"--n-icon-margin-left":j.left,"--n-icon-size":C,"--n-close-size":D,"--n-close-icon-size":v,"--n-close-border-radius":f,"--n-close-color-hover":p,"--n-close-color-pressed":m,"--n-close-icon-color":h,"--n-close-icon-color-hover":g,"--n-close-icon-color-pressed":_,"--n-color":d,"--n-text-color":l,"--n-border-radius":y,"--n-padding":S,"--n-line-height":a,"--n-border":s,"--n-content-margin":T,"--n-title-font-size":x,"--n-title-font-weight":b,"--n-title-text-color":c,"--n-action-space":w}}),f=r?b(`dialog`,m(()=>`${e.type[0]}${o.value[0]}`),d,e):void 0;return{mergedClsPrefix:n,rtlEnabled:a,mergedIconPlacement:o,mergedTheme:u,handlePositiveClick:s,handleNegativeClick:c,handleCloseClick:l,cssVars:r?void 0:d,themeClass:f?.themeClass,onRender:f?.onRender}},render(){var e;let{bordered:t,mergedIconPlacement:n,cssVars:r,closable:i,showIcon:a,title:o,content:c,action:l,negativeText:u,positiveText:d,positiveButtonProps:f,negativeButtonProps:p,handlePositiveClick:m,handleNegativeClick:h,mergedTheme:g,loading:_,type:v,mergedClsPrefix:y}=this;(e=this.onRender)==null||e.call(this);let b=a?s(le,{clsPrefix:y,class:`${y}-dialog__icon`},{default:()=>N(this.$slots.icon,e=>e||(this.icon?H(this.icon):tt[this.type]()))}):null,x=N(this.$slots.action,e=>e||d||u||l?s(`div`,{class:[`${y}-dialog__action`,this.actionClass],style:this.actionStyle},e||(l?[H(l)]:[this.negativeText&&s(ke,Object.assign({theme:g.peers.Button,themeOverrides:g.peerOverrides.Button,ghost:!0,size:`small`,onClick:h},p),{default:()=>H(this.negativeText)}),this.positiveText&&s(ke,Object.assign({theme:g.peers.Button,themeOverrides:g.peerOverrides.Button,size:`small`,type:v===`default`?`primary`:v,disabled:_,loading:_,onClick:m},f),{default:()=>H(this.positiveText)})])):null);return s(`div`,{class:[`${y}-dialog`,this.themeClass,this.closable&&`${y}-dialog--closable`,`${y}-dialog--icon-${n}`,t&&`${y}-dialog--bordered`,this.rtlEnabled&&`${y}-dialog--rtl`],style:r,role:`dialog`},i?N(this.$slots.close,e=>{let t=[`${y}-dialog__close`,this.rtlEnabled&&`${y}-dialog--rtl`];return e?s(`div`,{class:t},e):s(be,{focusable:this.closeFocusable,clsPrefix:y,class:t,onClick:this.handleCloseClick})}):null,a&&n===`top`?s(`div`,{class:`${y}-dialog-icon-container`},b):null,s(`div`,{class:[`${y}-dialog__title`,this.titleClass],style:this.titleStyle},a&&n===`left`?b:null,ne(this.$slots.header,()=>[H(o)])),s(`div`,{class:[`${y}-dialog__content`,x?``:`${y}-dialog__content--last`,this.contentClass],style:this.contentStyle},ne(this.$slots.default,()=>[H(c)])),x)}});function rt(e){let{modalColor:t,textColor2:n,boxShadow3:r}=e;return{color:t,textColor:n,boxShadow:r}}var it=k({name:`Modal`,common:ee,peers:{Scrollbar:oe,Dialog:Qe,Card:Ae},self:rt}),at=y(`n-modal-provider`),ot=y(`n-modal-api`),st=y(`n-modal-reactive-list`);function ct(){let e=g(ot,null);return e===null&&S(`use-modal`,`No outer <n-modal-provider /> founded.`),e}function lt(){let e=g(st,null);return e===null&&S(`use-modal-reactive-list`,`No outer <n-modal-provider /> founded.`),e}var Z=`n-draggable`;function ut(e,t){let n,r=m(()=>e.value!==!1),i=m(()=>r.value?Z:``),a=m(()=>{let t=e.value;return t===!0||t===!1?!0:t?t.bounds!==`none`:!0});function s(e){let r=e.querySelector(`.${Z}`);if(!r||!i.value)return;let o=0,s=0,c=0,l=0,u=0,d=0,f,p=null,m=null;function h(t){t.preventDefault(),f=t;let{x:n,y:r,right:i,bottom:a}=e.getBoundingClientRect();s=n,l=r,o=window.innerWidth-i,c=window.innerHeight-a;let{left:p,top:m}=e.style;u=+m.slice(0,-2),d=+p.slice(0,-2)}function g(){m&&=(e.style.top=`${m.y}px`,e.style.left=`${m.x}px`,null),p=null}function _(e){if(!f)return;let{clientX:t,clientY:n}=f,r=e.clientX-t,i=e.clientY-n;a.value&&(r>o?r=o:-r>s&&(r=-s),i>c?i=c:-i>l&&(i=-l)),m={x:r+d,y:i+u},p||=requestAnimationFrame(g)}function v(){f=void 0,p&&=(cancelAnimationFrame(p),null),m&&=(e.style.top=`${m.y}px`,e.style.left=`${m.x}px`,null),t.onEnd(e)}F(`mousedown`,r,h),F(`mousemove`,window,_),F(`mouseup`,window,v),n=()=>{p&&cancelAnimationFrame(p),I(`mousedown`,r,h),I(`mousemove`,window,_),I(`mouseup`,window,v)}}function c(){n&&=(n(),void 0)}return o(c),{stopDrag:c,startDrag:s,draggableRef:r,draggableClassRef:i}}var Q=Object.assign(Object.assign({},Me),X),dt=V(Q),ft=_({name:`ModalBody`,inheritAttrs:!1,slots:Object,props:Object.assign(Object.assign({show:{type:Boolean,required:!0},preset:String,displayDirective:{type:String,required:!0},trapFocus:{type:Boolean,default:!0},autoFocus:{type:Boolean,default:!0},blockScroll:Boolean,draggable:{type:[Boolean,Object],default:!1},maskHidden:Boolean},Q),{renderMask:Function,onClickoutside:Function,onBeforeLeave:{type:Function,required:!0},onAfterLeave:{type:Function,required:!0},onPositiveClick:{type:Function,required:!0},onNegativeClick:{type:Function,required:!0},onClose:{type:Function,required:!0},onAfterEnter:Function,onEsc:Function}),setup(e){let n=l(null),a=l(null),o=l(e.show),s=l(null),c=l(null),u=g(R),d=null;r(h(e,`show`),e=>{e&&(d=u.getMousePosition())},{immediate:!0});let{stopDrag:p,startDrag:_,draggableRef:v,draggableClassRef:y}=ut(h(e,`draggable`),{onEnd:e=>{C(e)}}),b=m(()=>f([e.titleClass,y.value])),x=m(()=>f([e.headerClass,y.value]));r(h(e,`show`),e=>{e&&(o.value=!0)}),Ye(m(()=>e.blockScroll&&o.value));function S(){if(u.transformOriginRef.value===`center`)return``;let{value:e}=s,{value:t}=c;return e===null||t===null?``:a.value?`${e}px ${t+a.value.containerScrollTop}px`:``}function C(e){if(u.transformOriginRef.value===`center`||!d||!a.value)return;let t=a.value.containerScrollTop,{offsetLeft:n,offsetTop:r}=e,i=d.y;s.value=-(n-d.x),c.value=-(r-i-t),e.style.transformOrigin=S()}function w(e){t(()=>{C(e)})}function T(t){t.style.transformOrigin=S(),e.onBeforeLeave()}function E(t){let n=t;v.value&&_(n),e.onAfterEnter&&e.onAfterEnter(n)}function D(){o.value=!1,s.value=null,c.value=null,p(),e.onAfterLeave()}function O(){let{onClose:t}=e;t&&t()}function k(){e.onNegativeClick()}function A(){e.onPositiveClick()}let j=l(null);return r(j,e=>{e&&t(()=>{let t=e.el;t&&n.value!==t&&(n.value=t)})}),i(pe,n),i(me,null),i(fe,null),{mergedTheme:u.mergedThemeRef,appear:u.appearRef,isMounted:u.isMountedRef,mergedClsPrefix:u.mergedClsPrefixRef,bodyRef:n,scrollbarRef:a,draggableClass:y,displayed:o,childNodeRef:j,cardHeaderClass:x,dialogTitleClass:b,handlePositiveClick:A,handleNegativeClick:k,handleCloseClick:O,handleAfterEnter:E,handleAfterLeave:D,handleBeforeLeave:T,handleEnter:w}},render(){let{$slots:e,$attrs:t,handleEnter:r,handleAfterEnter:i,handleAfterLeave:a,handleBeforeLeave:o,preset:l,mergedClsPrefix:d}=this,f=null;if(!l){if(f=he(`default`,e.default,{draggableClass:this.draggableClass}),!f){te(`modal`,`default slot is empty`);return}f=u(f),f.props=n({class:`${d}-modal`},t,f.props||{})}return this.displayDirective===`show`||this.displayed||this.show?c(s(`div`,{role:`none`,class:[`${d}-modal-body-wrapper`,this.maskHidden&&`${d}-modal-body-wrapper--mask-hidden`]},s(se,{ref:`scrollbarRef`,theme:this.mergedTheme.peers.Scrollbar,themeOverrides:this.mergedTheme.peerOverrides.Scrollbar,contentClass:`${d}-modal-scroll-content`},{default:()=>[this.renderMask?.call(this),s(ue,{disabled:!this.trapFocus||this.maskHidden,active:this.show,onEsc:this.onEsc,autoFocus:this.autoFocus},{default:()=>s(C,{name:`fade-in-scale-up-transition`,appear:this.appear??this.isMounted,onEnter:r,onAfterEnter:i,onAfterLeave:a,onBeforeLeave:o},{default:()=>{let t=[[v,this.show]],{onClickoutside:n}=this;return n&&t.push([de,this.onClickoutside,void 0,{capture:!0}]),c(this.preset===`confirm`||this.preset===`dialog`?s(nt,Object.assign({},this.$attrs,{class:[`${d}-modal`,this.$attrs.class],ref:`bodyRef`,theme:this.mergedTheme.peers.Dialog,themeOverrides:this.mergedTheme.peerOverrides.Dialog},L(this.$props,$e),{titleClass:this.dialogTitleClass,"aria-modal":`true`}),e):this.preset===`card`?s(Ne,Object.assign({},this.$attrs,{ref:`bodyRef`,class:[`${d}-modal`,this.$attrs.class],theme:this.mergedTheme.peers.Card,themeOverrides:this.mergedTheme.peerOverrides.Card},L(this.$props,je),{headerClass:this.cardHeaderClass,"aria-modal":`true`,role:`dialog`}),e):this.childNodeRef=f,t)}})})]})),[[v,this.displayDirective===`if`||this.displayed||this.show]]):null}}),pt=M([T(`modal-container`,`
 position: fixed;
 left: 0;
 top: 0;
 height: 0;
 width: 0;
 display: flex;
 `),T(`modal-mask`,`
 position: fixed;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 background-color: rgba(0, 0, 0, .4);
 `,[De({enterDuration:`.25s`,leaveDuration:`.25s`,enterCubicBezier:`var(--n-bezier-ease-out)`,leaveCubicBezier:`var(--n-bezier-ease-out)`})]),T(`modal-body-wrapper`,`
 position: fixed;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 overflow: visible;
 `,[T(`modal-scroll-content`,`
 min-height: 100%;
 display: flex;
 position: relative;
 `),O(`mask-hidden`,`pointer-events: none;`,[T(`modal-scroll-content`,[M(`> *`,`
 pointer-events: all;
 `)])])]),T(`modal`,`
 position: relative;
 align-self: center;
 color: var(--n-text-color);
 margin: auto;
 box-shadow: var(--n-box-shadow);
 `,[ye({duration:`.25s`,enterScale:`.5`}),M(`.${Z}`,`
 cursor: move;
 user-select: none;
 `)])]),$=Object.assign(Object.assign(Object.assign(Object.assign({},j.props),{show:Boolean,showMask:{type:Boolean,default:!0},maskClosable:{type:Boolean,default:!0},preset:String,to:[String,Object],displayDirective:{type:String,default:`if`},transformOrigin:{type:String,default:`mouse`},zIndex:Number,autoFocus:{type:Boolean,default:!0},trapFocus:{type:Boolean,default:!0},closeOnEsc:{type:Boolean,default:!0},blockScroll:{type:Boolean,default:!0}}),Q),{draggable:[Boolean,Object],onEsc:Function,"onUpdate:show":[Function,Array],onUpdateShow:[Function,Array],onAfterEnter:Function,onBeforeLeave:Function,onAfterLeave:Function,onClose:Function,onPositiveClick:Function,onNegativeClick:Function,onMaskClick:Function,internalDialog:Boolean,internalModal:Boolean,internalAppear:{type:Boolean,default:void 0},overlayStyle:[String,Object],onBeforeHide:Function,onAfterHide:Function,onHide:Function,unstableShowMask:{type:Boolean,default:void 0}}),mt=_({name:`Modal`,inheritAttrs:!1,props:$,slots:Object,setup(e){let t=l(null),{mergedClsPrefixRef:n,namespaceRef:r,inlineThemeDisabled:a}=x(e),o=j(`Modal`,`-modal`,pt,it,e,n),s=Ve(64),c=Le(),u=re(),d=e.internalDialog?g(Pe,null):null,f=e.internalModal?g(ge,null):null,p=We();function _(t){let{onUpdateShow:n,"onUpdate:show":r,onHide:i}=e;n&&P(n,t),r&&P(r,t),i&&!t&&i(t)}function v(){let{onClose:t}=e;t?Promise.resolve(t()).then(e=>{e!==!1&&_(!1)}):_(!1)}function y(){let{onPositiveClick:t}=e;t?Promise.resolve(t()).then(e=>{e!==!1&&_(!1)}):_(!1)}function S(){let{onNegativeClick:t}=e;t?Promise.resolve(t()).then(e=>{e!==!1&&_(!1)}):_(!1)}function C(){let{onBeforeLeave:t,onBeforeHide:n}=e;t&&P(t),n&&n()}function w(){let{onAfterLeave:t,onAfterHide:n}=e;t&&P(t),n&&n()}function T(n){let{onMaskClick:r}=e;r&&r(n),e.maskClosable&&t.value?.contains(ae(n))&&_(!1)}function E(t){var n;(n=e.onEsc)==null||n.call(e),e.show&&e.closeOnEsc&&Se(t)&&(p.value||_(!1))}i(R,{getMousePosition:()=>{let e=d||f;if(e){let{clickedRef:t,clickedPositionRef:n}=e;if(t.value&&n.value)return n.value}return s.value?c.value:null},mergedClsPrefixRef:n,mergedThemeRef:o,isMountedRef:u,appearRef:h(e,`internalAppear`),transformOriginRef:h(e,`transformOrigin`)});let D=m(()=>{let{common:{cubicBezierEaseOut:e},self:{boxShadow:t,color:n,textColor:r}}=o.value;return{"--n-bezier-ease-out":e,"--n-box-shadow":t,"--n-color":n,"--n-text-color":r}}),O=a?b(`theme-class`,void 0,D,e):void 0;return{mergedClsPrefix:n,namespace:r,isMounted:u,containerRef:t,presetProps:m(()=>L(e,dt)),handleEsc:E,handleAfterLeave:w,handleClickoutside:T,handleBeforeLeave:C,doUpdateShow:_,handleNegativeClick:S,handlePositiveClick:y,handleCloseClick:v,cssVars:a?void 0:D,themeClass:O?.themeClass,onRender:O?.onRender}},render(){let{mergedClsPrefix:e}=this;return s(_e,{to:this.to,show:this.show},{default:()=>{var t;(t=this.onRender)==null||t.call(this);let{showMask:n}=this;return c(s(`div`,{role:`none`,ref:`containerRef`,class:[`${e}-modal-container`,this.themeClass,this.namespace],style:this.cssVars},s(ft,Object.assign({style:this.overlayStyle},this.$attrs,{ref:`bodyWrapper`,displayDirective:this.displayDirective,show:this.show,preset:this.preset,autoFocus:this.autoFocus,trapFocus:this.trapFocus,draggable:this.draggable,blockScroll:this.blockScroll,maskHidden:!n},this.presetProps,{onEsc:this.handleEsc,onClose:this.handleCloseClick,onNegativeClick:this.handleNegativeClick,onPositiveClick:this.handlePositiveClick,onBeforeLeave:this.handleBeforeLeave,onAfterEnter:this.onAfterEnter,onAfterLeave:this.handleAfterLeave,onClickoutside:n?void 0:this.handleClickoutside,renderMask:n?()=>s(C,{name:`fade-in-transition`,key:`mask`,appear:this.internalAppear??this.isMounted},{default:()=>this.show?s(`div`,{"aria-hidden":!0,ref:`containerRef`,class:`${e}-modal-mask`,onClick:this.handleClickoutside}):null}):void 0}),this.$slots)),[[ve,{zIndex:this.zIndex,enabled:this.show}]])}})}});export{Le as _,ot as a,rt as c,X as d,Ze as f,Ve as g,We as h,lt as i,nt as l,Ye as m,$ as n,at as o,Y as p,ct as r,st as s,mt as t,$e as u};