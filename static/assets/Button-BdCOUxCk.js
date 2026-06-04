import{A as e,D as t,S as n,at as r,l as i,lt as a,w as o,y as s}from"./runtime-core.esm-bundler-jDnCq53A.js";import{H as c,N as l,P as u,U as d,W as f,a as p,dt as m,gt as h,ht as g,mt as _,pt as v,r as y,t as b,ut as x}from"./light-Vq2WEi6e.js";import{a as S,c as C,d as w,i as T,o as E,p as D,r as O,t as k}from"./Loading-C7sDZO8k.js";import{n as A,r as j}from"./use-form-item-DnBktFtM.js";import{t as M}from"./is-browser-DrnB8jUw.js";import{t as N}from"./color-to-class-CShJra4W.js";import{t as P}from"./FadeInExpandTransition-Cpyebht7.js";var{cubicBezierEaseInOut:F}=p;function I({duration:e=`.2s`,delay:t=`.1s`}={}){return[x(`&.fade-in-width-expand-transition-leave-from, &.fade-in-width-expand-transition-enter-to`,{opacity:1}),x(`&.fade-in-width-expand-transition-leave-to, &.fade-in-width-expand-transition-enter-from`,`
 opacity: 0!important;
 margin-left: 0!important;
 margin-right: 0!important;
 `),x(`&.fade-in-width-expand-transition-leave-active`,`
 overflow: hidden;
 transition:
 opacity ${e} ${F},
 max-width ${e} ${F} ${t},
 margin-left ${e} ${F} ${t},
 margin-right ${e} ${F} ${t};
 `),x(`&.fade-in-width-expand-transition-enter-active`,`
 overflow: hidden;
 transition:
 opacity ${e} ${F} ${t},
 max-width ${e} ${F},
 margin-left ${e} ${F},
 margin-right ${e} ${F};
 `)]}var L=m(`base-wave`,`
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border-radius: inherit;
`),R=s({name:`BaseWave`,props:{clsPrefix:{type:String,required:!0}},setup(n){S(`-base-wave`,L,a(n,`clsPrefix`));let i=r(null),o=r(!1),s=null;return e(()=>{s!==null&&window.clearTimeout(s)}),{active:o,selfRef:i,play(){s!==null&&(window.clearTimeout(s),o.value=!1,s=null),t(()=>{var e;(e=i.value)==null||e.offsetHeight,o.value=!0,s=window.setTimeout(()=>{o.value=!1,s=null},1e3)})}}},render(){let{clsPrefix:e}=this;return n(`div`,{ref:`selfRef`,"aria-hidden":!0,class:[`${e}-base-wave`,this.active&&`${e}-base-wave--active`]})}}),z=M&&`chrome`in window;M&&navigator.userAgent.includes(`Firefox`);var B=M&&navigator.userAgent.includes(`Safari`)&&!z;function V(e){return f(e,[255,255,255,.16])}function H(e){return f(e,[0,0,0,.12])}var U=c(`n-button-group`),W={paddingTiny:`0 6px`,paddingSmall:`0 10px`,paddingMedium:`0 14px`,paddingLarge:`0 18px`,paddingRoundTiny:`0 10px`,paddingRoundSmall:`0 14px`,paddingRoundMedium:`0 18px`,paddingRoundLarge:`0 22px`,iconMarginTiny:`6px`,iconMarginSmall:`6px`,iconMarginMedium:`6px`,iconMarginLarge:`6px`,iconSizeTiny:`14px`,iconSizeSmall:`18px`,iconSizeMedium:`18px`,iconSizeLarge:`20px`,rippleDuration:`.6s`};function G(e){let{heightTiny:t,heightSmall:n,heightMedium:r,heightLarge:i,borderRadius:a,fontSizeTiny:o,fontSizeSmall:s,fontSizeMedium:c,fontSizeLarge:l,opacityDisabled:u,textColor2:d,textColor3:f,primaryColorHover:p,primaryColorPressed:m,borderColor:h,primaryColor:g,baseColor:_,infoColor:v,infoColorHover:y,infoColorPressed:b,successColor:x,successColorHover:S,successColorPressed:C,warningColor:w,warningColorHover:T,warningColorPressed:E,errorColor:D,errorColorHover:O,errorColorPressed:k,fontWeight:A,buttonColor2:j,buttonColor2Hover:M,buttonColor2Pressed:N,fontWeightStrong:P}=e;return Object.assign(Object.assign({},W),{heightTiny:t,heightSmall:n,heightMedium:r,heightLarge:i,borderRadiusTiny:a,borderRadiusSmall:a,borderRadiusMedium:a,borderRadiusLarge:a,fontSizeTiny:o,fontSizeSmall:s,fontSizeMedium:c,fontSizeLarge:l,opacityDisabled:u,colorOpacitySecondary:`0.16`,colorOpacitySecondaryHover:`0.22`,colorOpacitySecondaryPressed:`0.28`,colorSecondary:j,colorSecondaryHover:M,colorSecondaryPressed:N,colorTertiary:j,colorTertiaryHover:M,colorTertiaryPressed:N,colorQuaternary:`#0000`,colorQuaternaryHover:M,colorQuaternaryPressed:N,color:`#0000`,colorHover:`#0000`,colorPressed:`#0000`,colorFocus:`#0000`,colorDisabled:`#0000`,textColor:d,textColorTertiary:f,textColorHover:p,textColorPressed:m,textColorFocus:p,textColorDisabled:d,textColorText:d,textColorTextHover:p,textColorTextPressed:m,textColorTextFocus:p,textColorTextDisabled:d,textColorGhost:d,textColorGhostHover:p,textColorGhostPressed:m,textColorGhostFocus:p,textColorGhostDisabled:d,border:`1px solid ${h}`,borderHover:`1px solid ${p}`,borderPressed:`1px solid ${m}`,borderFocus:`1px solid ${p}`,borderDisabled:`1px solid ${h}`,rippleColor:g,colorPrimary:g,colorHoverPrimary:p,colorPressedPrimary:m,colorFocusPrimary:p,colorDisabledPrimary:g,textColorPrimary:_,textColorHoverPrimary:_,textColorPressedPrimary:_,textColorFocusPrimary:_,textColorDisabledPrimary:_,textColorTextPrimary:g,textColorTextHoverPrimary:p,textColorTextPressedPrimary:m,textColorTextFocusPrimary:p,textColorTextDisabledPrimary:d,textColorGhostPrimary:g,textColorGhostHoverPrimary:p,textColorGhostPressedPrimary:m,textColorGhostFocusPrimary:p,textColorGhostDisabledPrimary:g,borderPrimary:`1px solid ${g}`,borderHoverPrimary:`1px solid ${p}`,borderPressedPrimary:`1px solid ${m}`,borderFocusPrimary:`1px solid ${p}`,borderDisabledPrimary:`1px solid ${g}`,rippleColorPrimary:g,colorInfo:v,colorHoverInfo:y,colorPressedInfo:b,colorFocusInfo:y,colorDisabledInfo:v,textColorInfo:_,textColorHoverInfo:_,textColorPressedInfo:_,textColorFocusInfo:_,textColorDisabledInfo:_,textColorTextInfo:v,textColorTextHoverInfo:y,textColorTextPressedInfo:b,textColorTextFocusInfo:y,textColorTextDisabledInfo:d,textColorGhostInfo:v,textColorGhostHoverInfo:y,textColorGhostPressedInfo:b,textColorGhostFocusInfo:y,textColorGhostDisabledInfo:v,borderInfo:`1px solid ${v}`,borderHoverInfo:`1px solid ${y}`,borderPressedInfo:`1px solid ${b}`,borderFocusInfo:`1px solid ${y}`,borderDisabledInfo:`1px solid ${v}`,rippleColorInfo:v,colorSuccess:x,colorHoverSuccess:S,colorPressedSuccess:C,colorFocusSuccess:S,colorDisabledSuccess:x,textColorSuccess:_,textColorHoverSuccess:_,textColorPressedSuccess:_,textColorFocusSuccess:_,textColorDisabledSuccess:_,textColorTextSuccess:x,textColorTextHoverSuccess:S,textColorTextPressedSuccess:C,textColorTextFocusSuccess:S,textColorTextDisabledSuccess:d,textColorGhostSuccess:x,textColorGhostHoverSuccess:S,textColorGhostPressedSuccess:C,textColorGhostFocusSuccess:S,textColorGhostDisabledSuccess:x,borderSuccess:`1px solid ${x}`,borderHoverSuccess:`1px solid ${S}`,borderPressedSuccess:`1px solid ${C}`,borderFocusSuccess:`1px solid ${S}`,borderDisabledSuccess:`1px solid ${x}`,rippleColorSuccess:x,colorWarning:w,colorHoverWarning:T,colorPressedWarning:E,colorFocusWarning:T,colorDisabledWarning:w,textColorWarning:_,textColorHoverWarning:_,textColorPressedWarning:_,textColorFocusWarning:_,textColorDisabledWarning:_,textColorTextWarning:w,textColorTextHoverWarning:T,textColorTextPressedWarning:E,textColorTextFocusWarning:T,textColorTextDisabledWarning:d,textColorGhostWarning:w,textColorGhostHoverWarning:T,textColorGhostPressedWarning:E,textColorGhostFocusWarning:T,textColorGhostDisabledWarning:w,borderWarning:`1px solid ${w}`,borderHoverWarning:`1px solid ${T}`,borderPressedWarning:`1px solid ${E}`,borderFocusWarning:`1px solid ${T}`,borderDisabledWarning:`1px solid ${w}`,rippleColorWarning:w,colorError:D,colorHoverError:O,colorPressedError:k,colorFocusError:O,colorDisabledError:D,textColorError:_,textColorHoverError:_,textColorPressedError:_,textColorFocusError:_,textColorDisabledError:_,textColorTextError:D,textColorTextHoverError:O,textColorTextPressedError:k,textColorTextFocusError:O,textColorTextDisabledError:d,textColorGhostError:D,textColorGhostHoverError:O,textColorGhostPressedError:k,textColorGhostFocusError:O,textColorGhostDisabledError:D,borderError:`1px solid ${D}`,borderHoverError:`1px solid ${O}`,borderPressedError:`1px solid ${k}`,borderFocusError:`1px solid ${O}`,borderDisabledError:`1px solid ${D}`,rippleColorError:D,waveOpacity:`0.6`,fontWeight:A,fontWeightStrong:P})}var K={name:`Button`,common:b,self:G},q=x([m(`button`,`
 margin: 0;
 font-weight: var(--n-font-weight);
 line-height: 1;
 font-family: inherit;
 padding: var(--n-padding);
 height: var(--n-height);
 font-size: var(--n-font-size);
 border-radius: var(--n-border-radius);
 color: var(--n-text-color);
 background-color: var(--n-color);
 width: var(--n-width);
 white-space: nowrap;
 outline: none;
 position: relative;
 z-index: auto;
 border: none;
 display: inline-flex;
 flex-wrap: nowrap;
 flex-shrink: 0;
 align-items: center;
 justify-content: center;
 user-select: none;
 -webkit-user-select: none;
 text-align: center;
 cursor: pointer;
 text-decoration: none;
 transition:
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 opacity .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[_(`color`,[v(`border`,{borderColor:`var(--n-border-color)`}),_(`disabled`,[v(`border`,{borderColor:`var(--n-border-color-disabled)`})]),g(`disabled`,[x(`&:focus`,[v(`state-border`,{borderColor:`var(--n-border-color-focus)`})]),x(`&:hover`,[v(`state-border`,{borderColor:`var(--n-border-color-hover)`})]),x(`&:active`,[v(`state-border`,{borderColor:`var(--n-border-color-pressed)`})]),_(`pressed`,[v(`state-border`,{borderColor:`var(--n-border-color-pressed)`})])])]),_(`disabled`,{backgroundColor:`var(--n-color-disabled)`,color:`var(--n-text-color-disabled)`},[v(`border`,{border:`var(--n-border-disabled)`})]),g(`disabled`,[x(`&:focus`,{backgroundColor:`var(--n-color-focus)`,color:`var(--n-text-color-focus)`},[v(`state-border`,{border:`var(--n-border-focus)`})]),x(`&:hover`,{backgroundColor:`var(--n-color-hover)`,color:`var(--n-text-color-hover)`},[v(`state-border`,{border:`var(--n-border-hover)`})]),x(`&:active`,{backgroundColor:`var(--n-color-pressed)`,color:`var(--n-text-color-pressed)`},[v(`state-border`,{border:`var(--n-border-pressed)`})]),_(`pressed`,{backgroundColor:`var(--n-color-pressed)`,color:`var(--n-text-color-pressed)`},[v(`state-border`,{border:`var(--n-border-pressed)`})])]),_(`loading`,`cursor: wait;`),m(`base-wave`,`
 pointer-events: none;
 top: 0;
 right: 0;
 bottom: 0;
 left: 0;
 animation-iteration-count: 1;
 animation-duration: var(--n-ripple-duration);
 animation-timing-function: var(--n-bezier-ease-out), var(--n-bezier-ease-out);
 `,[_(`active`,{zIndex:1,animationName:`button-wave-spread, button-wave-opacity`})]),M&&`MozBoxSizing`in document.createElement(`div`).style?x(`&::moz-focus-inner`,{border:0}):null,v(`border, state-border`,`
 position: absolute;
 left: 0;
 top: 0;
 right: 0;
 bottom: 0;
 border-radius: inherit;
 transition: border-color .3s var(--n-bezier);
 pointer-events: none;
 `),v(`border`,`
 border: var(--n-border);
 `),v(`state-border`,`
 border: var(--n-border);
 border-color: #0000;
 z-index: 1;
 `),v(`icon`,`
 margin: var(--n-icon-margin);
 margin-left: 0;
 height: var(--n-icon-size);
 width: var(--n-icon-size);
 max-width: var(--n-icon-size);
 font-size: var(--n-icon-size);
 position: relative;
 flex-shrink: 0;
 `,[m(`icon-slot`,`
 height: var(--n-icon-size);
 width: var(--n-icon-size);
 position: absolute;
 left: 0;
 top: 50%;
 transform: translateY(-50%);
 display: flex;
 align-items: center;
 justify-content: center;
 `,[O({top:`50%`,originalTransform:`translateY(-50%)`})]),I()]),v(`content`,`
 display: flex;
 align-items: center;
 flex-wrap: nowrap;
 min-width: 0;
 `,[x(`~`,[v(`icon`,{margin:`var(--n-icon-margin)`,marginRight:0})])]),_(`block`,`
 display: flex;
 width: 100%;
 `),_(`dashed`,[v(`border, state-border`,{borderStyle:`dashed !important`})]),_(`disabled`,{cursor:`not-allowed`,opacity:`var(--n-opacity-disabled)`})]),x(`@keyframes button-wave-spread`,{from:{boxShadow:`0 0 0.5px 0 var(--n-ripple-color)`},to:{boxShadow:`0 0 0.5px 4.5px var(--n-ripple-color)`}}),x(`@keyframes button-wave-opacity`,{from:{opacity:`var(--n-wave-opacity)`},to:{opacity:0}})]),J=Object.assign(Object.assign({},y.props),{color:String,textColor:String,text:Boolean,block:Boolean,loading:Boolean,disabled:Boolean,circle:Boolean,size:String,ghost:Boolean,round:Boolean,secondary:Boolean,tertiary:Boolean,quaternary:Boolean,strong:Boolean,focusable:{type:Boolean,default:!0},keyboard:{type:Boolean,default:!0},tag:{type:String,default:`button`},type:{type:String,default:`default`},dashed:Boolean,renderIcon:Function,iconPlacement:{type:String,default:`left`},attrType:{type:String,default:`button`},bordered:{type:Boolean,default:!0},onClick:[Function,Array],nativeFocusBehavior:{type:Boolean,default:!B},spinProps:Object}),Y=s({name:`Button`,props:J,slots:Object,setup(e){let t=r(null),n=r(null),a=r(!1),s=j(()=>!e.quaternary&&!e.tertiary&&!e.secondary&&!e.text&&(!e.color||e.ghost||e.dashed)&&e.bordered),c=o(U,{}),{inlineThemeDisabled:f,mergedClsPrefixRef:p,mergedRtlRef:m,mergedComponentPropsRef:g}=u(e),{mergedSizeRef:_}=A({},{defaultSize:`medium`,mergedSize:t=>{let{size:n}=e;if(n)return n;let{size:r}=c;if(r)return r;let{mergedSize:i}=t||{};return i?i.value:g?.value?.Button?.size||`medium`}}),v=i(()=>e.focusable&&!e.disabled),b=n=>{var r;v.value||n.preventDefault(),!e.nativeFocusBehavior&&(n.preventDefault(),!e.disabled&&v.value&&((r=t.value)==null||r.focus({preventScroll:!0})))},x=t=>{var r;if(!e.disabled&&!e.loading){let{onClick:i}=e;i&&D(i,t),e.text||(r=n.value)==null||r.play()}},S=t=>{switch(t.key){case`Enter`:if(!e.keyboard)return;a.value=!1}},C=t=>{switch(t.key){case`Enter`:if(!e.keyboard||e.loading){t.preventDefault();return}a.value=!0}},w=()=>{a.value=!1},T=y(`Button`,`-button`,q,K,e,p),O=E(`Button`,m,p),k=i(()=>{let{common:{cubicBezierEaseInOut:t,cubicBezierEaseOut:n},self:r}=T.value,{rippleDuration:i,opacityDisabled:a,fontWeight:o,fontWeightStrong:s}=r,c=_.value,{dashed:l,type:u,ghost:f,text:p,color:m,round:g,circle:v,textColor:y,secondary:b,tertiary:x,quaternary:S,strong:C}=e,w={"--n-font-weight":C?s:o},E={"--n-color":`initial`,"--n-color-hover":`initial`,"--n-color-pressed":`initial`,"--n-color-focus":`initial`,"--n-color-disabled":`initial`,"--n-ripple-color":`initial`,"--n-text-color":`initial`,"--n-text-color-hover":`initial`,"--n-text-color-pressed":`initial`,"--n-text-color-focus":`initial`,"--n-text-color-disabled":`initial`},D=u===`tertiary`,O=u===`default`,k=D?`default`:u;if(p){let e=y||m,t=e||r[h(`textColorText`,k)];E={"--n-color":`#0000`,"--n-color-hover":`#0000`,"--n-color-pressed":`#0000`,"--n-color-focus":`#0000`,"--n-color-disabled":`#0000`,"--n-ripple-color":`#0000`,"--n-text-color":t,"--n-text-color-hover":e?V(e):r[h(`textColorTextHover`,k)],"--n-text-color-pressed":e?H(e):r[h(`textColorTextPressed`,k)],"--n-text-color-focus":e?V(e):r[h(`textColorTextHover`,k)],"--n-text-color-disabled":e||r[h(`textColorTextDisabled`,k)]}}else if(f||l){let e=y||m;E={"--n-color":`#0000`,"--n-color-hover":`#0000`,"--n-color-pressed":`#0000`,"--n-color-focus":`#0000`,"--n-color-disabled":`#0000`,"--n-ripple-color":m||r[h(`rippleColor`,k)],"--n-text-color":e||r[h(`textColorGhost`,k)],"--n-text-color-hover":e?V(e):r[h(`textColorGhostHover`,k)],"--n-text-color-pressed":e?H(e):r[h(`textColorGhostPressed`,k)],"--n-text-color-focus":e?V(e):r[h(`textColorGhostHover`,k)],"--n-text-color-disabled":e||r[h(`textColorGhostDisabled`,k)]}}else if(b){let e=O?r.textColor:D?r.textColorTertiary:r[h(`color`,k)],t=m||e,n=u!==`default`&&u!==`tertiary`;E={"--n-color":n?d(t,{alpha:Number(r.colorOpacitySecondary)}):r.colorSecondary,"--n-color-hover":n?d(t,{alpha:Number(r.colorOpacitySecondaryHover)}):r.colorSecondaryHover,"--n-color-pressed":n?d(t,{alpha:Number(r.colorOpacitySecondaryPressed)}):r.colorSecondaryPressed,"--n-color-focus":n?d(t,{alpha:Number(r.colorOpacitySecondaryHover)}):r.colorSecondaryHover,"--n-color-disabled":r.colorSecondary,"--n-ripple-color":`#0000`,"--n-text-color":t,"--n-text-color-hover":t,"--n-text-color-pressed":t,"--n-text-color-focus":t,"--n-text-color-disabled":t}}else if(x||S){let e=O?r.textColor:D?r.textColorTertiary:r[h(`color`,k)],t=m||e;x?(E[`--n-color`]=r.colorTertiary,E[`--n-color-hover`]=r.colorTertiaryHover,E[`--n-color-pressed`]=r.colorTertiaryPressed,E[`--n-color-focus`]=r.colorSecondaryHover,E[`--n-color-disabled`]=r.colorTertiary):(E[`--n-color`]=r.colorQuaternary,E[`--n-color-hover`]=r.colorQuaternaryHover,E[`--n-color-pressed`]=r.colorQuaternaryPressed,E[`--n-color-focus`]=r.colorQuaternaryHover,E[`--n-color-disabled`]=r.colorQuaternary),E[`--n-ripple-color`]=`#0000`,E[`--n-text-color`]=t,E[`--n-text-color-hover`]=t,E[`--n-text-color-pressed`]=t,E[`--n-text-color-focus`]=t,E[`--n-text-color-disabled`]=t}else E={"--n-color":m||r[h(`color`,k)],"--n-color-hover":m?V(m):r[h(`colorHover`,k)],"--n-color-pressed":m?H(m):r[h(`colorPressed`,k)],"--n-color-focus":m?V(m):r[h(`colorFocus`,k)],"--n-color-disabled":m||r[h(`colorDisabled`,k)],"--n-ripple-color":m||r[h(`rippleColor`,k)],"--n-text-color":y||(m?r.textColorPrimary:D?r.textColorTertiary:r[h(`textColor`,k)]),"--n-text-color-hover":y||(m?r.textColorHoverPrimary:r[h(`textColorHover`,k)]),"--n-text-color-pressed":y||(m?r.textColorPressedPrimary:r[h(`textColorPressed`,k)]),"--n-text-color-focus":y||(m?r.textColorFocusPrimary:r[h(`textColorFocus`,k)]),"--n-text-color-disabled":y||(m?r.textColorDisabledPrimary:r[h(`textColorDisabled`,k)])};let A={"--n-border":`initial`,"--n-border-hover":`initial`,"--n-border-pressed":`initial`,"--n-border-focus":`initial`,"--n-border-disabled":`initial`};A=p?{"--n-border":`none`,"--n-border-hover":`none`,"--n-border-pressed":`none`,"--n-border-focus":`none`,"--n-border-disabled":`none`}:{"--n-border":r[h(`border`,k)],"--n-border-hover":r[h(`borderHover`,k)],"--n-border-pressed":r[h(`borderPressed`,k)],"--n-border-focus":r[h(`borderFocus`,k)],"--n-border-disabled":r[h(`borderDisabled`,k)]};let{[h(`height`,c)]:j,[h(`fontSize`,c)]:M,[h(`padding`,c)]:N,[h(`paddingRound`,c)]:P,[h(`iconSize`,c)]:F,[h(`borderRadius`,c)]:I,[h(`iconMargin`,c)]:L,waveOpacity:R}=r,z={"--n-width":v&&!p?j:`initial`,"--n-height":p?`initial`:j,"--n-font-size":M,"--n-padding":v||p?`initial`:g?P:N,"--n-icon-size":F,"--n-icon-margin":L,"--n-border-radius":p?`initial`:v||g?j:I};return Object.assign(Object.assign(Object.assign(Object.assign({"--n-bezier":t,"--n-bezier-ease-out":n,"--n-ripple-duration":i,"--n-opacity-disabled":a,"--n-wave-opacity":R},w),E),A),z)}),M=f?l(`button`,i(()=>{let t=``,{dashed:n,type:r,ghost:i,text:a,color:o,round:s,circle:c,textColor:l,secondary:u,tertiary:d,quaternary:f,strong:p}=e;n&&(t+=`a`),i&&(t+=`b`),a&&(t+=`c`),s&&(t+=`d`),c&&(t+=`e`),u&&(t+=`f`),d&&(t+=`g`),f&&(t+=`h`),p&&(t+=`i`),o&&(t+=`j${N(o)}`),l&&(t+=`k${N(l)}`);let{value:m}=_;return t+=`l${m[0]}`,t+=`m${r[0]}`,t}),k,e):void 0;return{selfElRef:t,waveElRef:n,mergedClsPrefix:p,mergedFocusable:v,mergedSize:_,showBorder:s,enterPressed:a,rtlEnabled:O,handleMousedown:b,handleKeydown:C,handleBlur:w,handleKeyup:S,handleClick:x,customColorCssVars:i(()=>{let{color:t}=e;if(!t)return null;let n=V(t);return{"--n-border-color":t,"--n-border-color-hover":n,"--n-border-color-pressed":H(t),"--n-border-color-focus":n,"--n-border-color-disabled":t}}),cssVars:f?void 0:k,themeClass:M?.themeClass,onRender:M?.onRender}},render(){let{mergedClsPrefix:e,tag:t,onRender:r}=this;r?.();let i=w(this.$slots.default,t=>t&&n(`span`,{class:`${e}-button__content`},t));return n(t,{ref:`selfElRef`,class:[this.themeClass,`${e}-button`,`${e}-button--${this.type}-type`,`${e}-button--${this.mergedSize}-type`,this.rtlEnabled&&`${e}-button--rtl`,this.disabled&&`${e}-button--disabled`,this.block&&`${e}-button--block`,this.enterPressed&&`${e}-button--pressed`,!this.text&&this.dashed&&`${e}-button--dashed`,this.color&&`${e}-button--color`,this.secondary&&`${e}-button--secondary`,this.loading&&`${e}-button--loading`,this.ghost&&`${e}-button--ghost`],tabindex:this.mergedFocusable?0:-1,type:this.attrType,style:this.cssVars,disabled:this.disabled,onClick:this.handleClick,onBlur:this.handleBlur,onMousedown:this.handleMousedown,onKeyup:this.handleKeyup,onKeydown:this.handleKeydown},this.iconPlacement===`right`&&i,n(P,{width:!0},{default:()=>w(this.$slots.icon,t=>(this.loading||this.renderIcon||t)&&n(`span`,{class:`${e}-button__icon`,style:{margin:C(this.$slots.default)?`0`:``}},n(T,null,{default:()=>this.loading?n(k,Object.assign({clsPrefix:e,key:`loading`,class:`${e}-icon-slot`,strokeWidth:20},this.spinProps)):n(`div`,{key:`icon`,class:`${e}-icon-slot`,role:`none`},this.renderIcon?this.renderIcon():t)})))}),this.iconPlacement===`left`&&i,this.text?null:n(R,{ref:`waveElRef`,clsPrefix:e}),this.showBorder?n(`div`,{"aria-hidden":!0,class:`${e}-button__border`,style:this.customColorCssVars}):null,this.showBorder?n(`div`,{"aria-hidden":!0,class:`${e}-button__state-border`,style:this.customColorCssVars}):null)}}),X=Y;export{G as a,R as c,K as i,I as l,X as n,U as o,J as r,B as s,Y as t};