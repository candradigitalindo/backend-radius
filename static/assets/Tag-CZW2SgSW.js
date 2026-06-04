import{L as e,S as t,at as n,l as r,lt as i,y as a}from"./runtime-core.esm-bundler-jDnCq53A.js";import{H as o,N as s,P as c,U as l,dt as u,gt as d,ht as f,mt as p,pt as m,r as h,t as g,ut as _}from"./light-Vq2WEi6e.js";import{d as v,o as y,p as b}from"./Loading-C7sDZO8k.js";import{l as x}from"./replaceable-BKB0h89t.js";import{t as S}from"./Close-E-jUOvhm.js";import{t as C}from"./color-to-class-CShJra4W.js";var w={closeIconSizeTiny:`12px`,closeIconSizeSmall:`12px`,closeIconSizeMedium:`14px`,closeIconSizeLarge:`14px`,closeSizeTiny:`16px`,closeSizeSmall:`16px`,closeSizeMedium:`18px`,closeSizeLarge:`18px`,padding:`0 7px`,closeMargin:`0 0 0 4px`};function T(e){let{textColor2:t,primaryColorHover:n,primaryColorPressed:r,primaryColor:i,infoColor:a,successColor:o,warningColor:s,errorColor:c,baseColor:u,borderColor:d,opacityDisabled:f,tagColor:p,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,borderRadiusSmall:_,fontSizeMini:v,fontSizeTiny:y,fontSizeSmall:b,fontSizeMedium:x,heightMini:S,heightTiny:C,heightSmall:T,heightMedium:E,closeColorHover:D,closeColorPressed:O,buttonColor2Hover:k,buttonColor2Pressed:A,fontWeightStrong:j}=e;return Object.assign(Object.assign({},w),{closeBorderRadius:_,heightTiny:S,heightSmall:C,heightMedium:T,heightLarge:E,borderRadius:_,opacityDisabled:f,fontSizeTiny:v,fontSizeSmall:y,fontSizeMedium:b,fontSizeLarge:x,fontWeightStrong:j,textColorCheckable:t,textColorHoverCheckable:t,textColorPressedCheckable:t,textColorChecked:u,colorCheckable:`#0000`,colorHoverCheckable:k,colorPressedCheckable:A,colorChecked:i,colorCheckedHover:n,colorCheckedPressed:r,border:`1px solid ${d}`,textColor:t,color:p,colorBordered:`rgb(250, 250, 252)`,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,closeColorHover:D,closeColorPressed:O,borderPrimary:`1px solid ${l(i,{alpha:.3})}`,textColorPrimary:i,colorPrimary:l(i,{alpha:.12}),colorBorderedPrimary:l(i,{alpha:.1}),closeIconColorPrimary:i,closeIconColorHoverPrimary:i,closeIconColorPressedPrimary:i,closeColorHoverPrimary:l(i,{alpha:.12}),closeColorPressedPrimary:l(i,{alpha:.18}),borderInfo:`1px solid ${l(a,{alpha:.3})}`,textColorInfo:a,colorInfo:l(a,{alpha:.12}),colorBorderedInfo:l(a,{alpha:.1}),closeIconColorInfo:a,closeIconColorHoverInfo:a,closeIconColorPressedInfo:a,closeColorHoverInfo:l(a,{alpha:.12}),closeColorPressedInfo:l(a,{alpha:.18}),borderSuccess:`1px solid ${l(o,{alpha:.3})}`,textColorSuccess:o,colorSuccess:l(o,{alpha:.12}),colorBorderedSuccess:l(o,{alpha:.1}),closeIconColorSuccess:o,closeIconColorHoverSuccess:o,closeIconColorPressedSuccess:o,closeColorHoverSuccess:l(o,{alpha:.12}),closeColorPressedSuccess:l(o,{alpha:.18}),borderWarning:`1px solid ${l(s,{alpha:.35})}`,textColorWarning:s,colorWarning:l(s,{alpha:.15}),colorBorderedWarning:l(s,{alpha:.12}),closeIconColorWarning:s,closeIconColorHoverWarning:s,closeIconColorPressedWarning:s,closeColorHoverWarning:l(s,{alpha:.12}),closeColorPressedWarning:l(s,{alpha:.18}),borderError:`1px solid ${l(c,{alpha:.23})}`,textColorError:c,colorError:l(c,{alpha:.1}),colorBorderedError:l(c,{alpha:.08}),closeIconColorError:c,closeIconColorHoverError:c,closeIconColorPressedError:c,closeColorHoverError:l(c,{alpha:.12}),closeColorPressedError:l(c,{alpha:.18})})}var E={name:`Tag`,common:g,self:T},D={color:Object,type:{type:String,default:`default`},round:Boolean,size:String,closable:Boolean,disabled:{type:Boolean,default:void 0}},O=u(`tag`,`
 --n-close-margin: var(--n-close-margin-top) var(--n-close-margin-right) var(--n-close-margin-bottom) var(--n-close-margin-left);
 white-space: nowrap;
 position: relative;
 box-sizing: border-box;
 cursor: default;
 display: inline-flex;
 align-items: center;
 flex-wrap: nowrap;
 padding: var(--n-padding);
 border-radius: var(--n-border-radius);
 color: var(--n-text-color);
 background-color: var(--n-color);
 transition: 
 border-color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 opacity .3s var(--n-bezier);
 line-height: 1;
 height: var(--n-height);
 font-size: var(--n-font-size);
`,[p(`strong`,`
 font-weight: var(--n-font-weight-strong);
 `),m(`border`,`
 pointer-events: none;
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 border-radius: inherit;
 border: var(--n-border);
 transition: border-color .3s var(--n-bezier);
 `),m(`icon`,`
 display: flex;
 margin: 0 4px 0 0;
 color: var(--n-text-color);
 transition: color .3s var(--n-bezier);
 font-size: var(--n-avatar-size-override);
 `),m(`avatar`,`
 display: flex;
 margin: 0 6px 0 0;
 `),m(`close`,`
 margin: var(--n-close-margin);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `),p(`round`,`
 padding: 0 calc(var(--n-height) / 3);
 border-radius: calc(var(--n-height) / 2);
 `,[m(`icon`,`
 margin: 0 4px 0 calc((var(--n-height) - 8px) / -2);
 `),m(`avatar`,`
 margin: 0 6px 0 calc((var(--n-height) - 8px) / -2);
 `),p(`closable`,`
 padding: 0 calc(var(--n-height) / 4) 0 calc(var(--n-height) / 3);
 `)]),p(`icon, avatar`,[p(`round`,`
 padding: 0 calc(var(--n-height) / 3) 0 calc(var(--n-height) / 2);
 `)]),p(`disabled`,`
 cursor: not-allowed !important;
 opacity: var(--n-opacity-disabled);
 `),p(`checkable`,`
 cursor: pointer;
 box-shadow: none;
 color: var(--n-text-color-checkable);
 background-color: var(--n-color-checkable);
 `,[f(`disabled`,[_(`&:hover`,`background-color: var(--n-color-hover-checkable);`,[f(`checked`,`color: var(--n-text-color-hover-checkable);`)]),_(`&:active`,`background-color: var(--n-color-pressed-checkable);`,[f(`checked`,`color: var(--n-text-color-pressed-checkable);`)])]),p(`checked`,`
 color: var(--n-text-color-checked);
 background-color: var(--n-color-checked);
 `,[f(`disabled`,[_(`&:hover`,`background-color: var(--n-color-checked-hover);`),_(`&:active`,`background-color: var(--n-color-checked-pressed);`)])])])]),k=Object.assign(Object.assign(Object.assign({},h.props),D),{bordered:{type:Boolean,default:void 0},checked:Boolean,checkable:Boolean,strong:Boolean,triggerClickOnClose:Boolean,onClose:[Array,Function],onMouseenter:Function,onMouseleave:Function,"onUpdate:checked":Function,onUpdateChecked:Function,internalCloseFocusable:{type:Boolean,default:!0},internalCloseIsButtonTag:{type:Boolean,default:!0},onCheckedChange:Function}),A=o(`n-tag`),j=a({name:`Tag`,props:k,slots:Object,setup(t){let a=n(null),{mergedBorderedRef:o,mergedClsPrefixRef:l,inlineThemeDisabled:u,mergedRtlRef:f,mergedComponentPropsRef:p}=c(t),m=r(()=>t.size||p?.value?.Tag?.size||`medium`),g=h(`Tag`,`-tag`,O,E,t,l);e(A,{roundRef:i(t,`round`)});function _(){if(!t.disabled&&t.checkable){let{checked:e,onCheckedChange:n,onUpdateChecked:r,"onUpdate:checked":i}=t;r&&r(!e),i&&i(!e),n&&n(!e)}}function v(e){if(t.triggerClickOnClose||e.stopPropagation(),!t.disabled){let{onClose:n}=t;n&&b(n,e)}}let S={setTextContent(e){let{value:t}=a;t&&(t.textContent=e)}},w=y(`Tag`,f,l),T=r(()=>{let{type:e,color:{color:n,textColor:r}={}}=t,i=m.value,{common:{cubicBezierEaseInOut:a},self:{padding:s,closeMargin:c,borderRadius:l,opacityDisabled:u,textColorCheckable:f,textColorHoverCheckable:p,textColorPressedCheckable:h,textColorChecked:_,colorCheckable:v,colorHoverCheckable:y,colorPressedCheckable:b,colorChecked:S,colorCheckedHover:C,colorCheckedPressed:w,closeBorderRadius:T,fontWeightStrong:E,[d(`colorBordered`,e)]:D,[d(`closeSize`,i)]:O,[d(`closeIconSize`,i)]:k,[d(`fontSize`,i)]:A,[d(`height`,i)]:j,[d(`color`,e)]:M,[d(`textColor`,e)]:N,[d(`border`,e)]:P,[d(`closeIconColor`,e)]:F,[d(`closeIconColorHover`,e)]:I,[d(`closeIconColorPressed`,e)]:L,[d(`closeColorHover`,e)]:R,[d(`closeColorPressed`,e)]:z}}=g.value,B=x(c);return{"--n-font-weight-strong":E,"--n-avatar-size-override":`calc(${j} - 8px)`,"--n-bezier":a,"--n-border-radius":l,"--n-border":P,"--n-close-icon-size":k,"--n-close-color-pressed":z,"--n-close-color-hover":R,"--n-close-border-radius":T,"--n-close-icon-color":F,"--n-close-icon-color-hover":I,"--n-close-icon-color-pressed":L,"--n-close-icon-color-disabled":F,"--n-close-margin-top":B.top,"--n-close-margin-right":B.right,"--n-close-margin-bottom":B.bottom,"--n-close-margin-left":B.left,"--n-close-size":O,"--n-color":n||(o.value?D:M),"--n-color-checkable":v,"--n-color-checked":S,"--n-color-checked-hover":C,"--n-color-checked-pressed":w,"--n-color-hover-checkable":y,"--n-color-pressed-checkable":b,"--n-font-size":A,"--n-height":j,"--n-opacity-disabled":u,"--n-padding":s,"--n-text-color":r||N,"--n-text-color-checkable":f,"--n-text-color-checked":_,"--n-text-color-hover-checkable":p,"--n-text-color-pressed-checkable":h}}),D=u?s(`tag`,r(()=>{let e=``,{type:n,color:{color:r,textColor:i}={}}=t;return e+=n[0],e+=m.value[0],r&&(e+=`a${C(r)}`),i&&(e+=`b${C(i)}`),o.value&&(e+=`c`),e}),T,t):void 0;return Object.assign(Object.assign({},S),{rtlEnabled:w,mergedClsPrefix:l,contentRef:a,mergedBordered:o,handleClick:_,handleCloseClick:v,cssVars:u?void 0:T,themeClass:D?.themeClass,onRender:D?.onRender})},render(){var e;let{mergedClsPrefix:n,rtlEnabled:r,closable:i,color:{borderColor:a}={},round:o,onRender:s,$slots:c}=this;s?.();let l=v(c.avatar,e=>e&&t(`div`,{class:`${n}-tag__avatar`},e)),u=v(c.icon,e=>e&&t(`div`,{class:`${n}-tag__icon`},e));return t(`div`,{class:[`${n}-tag`,this.themeClass,{[`${n}-tag--rtl`]:r,[`${n}-tag--strong`]:this.strong,[`${n}-tag--disabled`]:this.disabled,[`${n}-tag--checkable`]:this.checkable,[`${n}-tag--checked`]:this.checkable&&this.checked,[`${n}-tag--round`]:o,[`${n}-tag--avatar`]:l,[`${n}-tag--icon`]:u,[`${n}-tag--closable`]:i}],style:this.cssVars,onClick:this.handleClick,onMouseenter:this.onMouseenter,onMouseleave:this.onMouseleave},u||l,t(`span`,{class:`${n}-tag__content`,ref:`contentRef`},(e=this.$slots).default?.call(e)),!this.checkable&&i?t(S,{clsPrefix:n,class:`${n}-tag__close`,disabled:this.disabled,onClick:this.handleCloseClick,focusable:this.internalCloseFocusable,round:o,isButtonTag:this.internalCloseIsButtonTag,absolute:!0}):null,!this.checkable&&this.mergedBordered?t(`div`,{class:`${n}-tag__border`,style:{borderColor:a}}):null)}});export{E as a,D as i,A as n,w as o,k as r,j as t};