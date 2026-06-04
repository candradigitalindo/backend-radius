import{S as e,l as t,y as n}from"./runtime-core.esm-bundler-jDnCq53A.js";import{N as r,P as i,_t as a,dt as o,gt as s,lt as c,mt as l,pt as u,r as d,t as f,ut as p,vt as m}from"./light-Vq2WEi6e.js";import{d as h,o as g,p as _,s as v}from"./Loading-C7sDZO8k.js";import{t as y}from"./Scrollbar-DmQk0fnf.js";import{l as b}from"./replaceable-BKB0h89t.js";import{t as x}from"./Close-E-jUOvhm.js";import{t as S}from"./keysOf-7YXhwv04.js";var C={paddingSmall:`12px 16px 12px`,paddingMedium:`19px 24px 20px`,paddingLarge:`23px 32px 24px`,paddingHuge:`27px 40px 28px`,titleFontSizeSmall:`16px`,titleFontSizeMedium:`18px`,titleFontSizeLarge:`18px`,titleFontSizeHuge:`18px`,closeIconSize:`18px`,closeSize:`22px`};function w(e){let{primaryColor:t,borderRadius:n,lineHeight:r,fontSize:i,cardColor:a,textColor2:o,textColor1:s,dividerColor:c,fontWeightStrong:l,closeIconColor:u,closeIconColorHover:d,closeIconColorPressed:f,closeColorHover:p,closeColorPressed:m,modalColor:h,boxShadow1:g,popoverColor:_,actionColor:v}=e;return Object.assign(Object.assign({},C),{lineHeight:r,color:a,colorModal:h,colorPopover:_,colorTarget:t,colorEmbedded:v,colorEmbeddedModal:v,colorEmbeddedPopover:v,textColor:o,titleTextColor:s,borderColor:c,actionColor:v,titleFontWeight:l,closeColorHover:p,closeColorPressed:m,closeBorderRadius:n,closeIconColor:u,closeIconColorHover:d,closeIconColorPressed:f,fontSizeSmall:i,fontSizeMedium:i,fontSizeLarge:i,fontSizeHuge:i,boxShadow:g,borderRadius:n})}var T={name:`Card`,common:f,self:w},E=o(`card-content`,`
 flex: 1;
 min-width: 0;
 box-sizing: border-box;
 padding: 0 var(--n-padding-left) var(--n-padding-bottom) var(--n-padding-left);
 font-size: var(--n-font-size);
`),D=p([o(`card`,`
 font-size: var(--n-font-size);
 line-height: var(--n-line-height);
 display: flex;
 flex-direction: column;
 width: 100%;
 box-sizing: border-box;
 position: relative;
 border-radius: var(--n-border-radius);
 background-color: var(--n-color);
 color: var(--n-text-color);
 word-break: break-word;
 transition: 
 color .3s var(--n-bezier),
 background-color .3s var(--n-bezier),
 box-shadow .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 `,[c({background:`var(--n-color-modal)`}),l(`hoverable`,[p(`&:hover`,`box-shadow: var(--n-box-shadow);`)]),l(`content-segmented`,[p(`>`,[o(`card-content`,`
 padding-top: var(--n-padding-bottom);
 `),u(`content-scrollbar`,[p(`>`,[o(`scrollbar-container`,[p(`>`,[o(`card-content`,`
 padding-top: var(--n-padding-bottom);
 `)])])])])])]),l(`content-soft-segmented`,[p(`>`,[o(`card-content`,`
 margin: 0 var(--n-padding-left);
 padding: var(--n-padding-bottom) 0;
 `),u(`content-scrollbar`,[p(`>`,[o(`scrollbar-container`,[p(`>`,[o(`card-content`,`
 margin: 0 var(--n-padding-left);
 padding: var(--n-padding-bottom) 0;
 `)])])])])])]),l(`footer-segmented`,[p(`>`,[u(`footer`,`
 padding-top: var(--n-padding-bottom);
 `)])]),l(`footer-soft-segmented`,[p(`>`,[u(`footer`,`
 padding: var(--n-padding-bottom) 0;
 margin: 0 var(--n-padding-left);
 `)])]),p(`>`,[o(`card-header`,`
 box-sizing: border-box;
 display: flex;
 align-items: center;
 font-size: var(--n-title-font-size);
 padding:
 var(--n-padding-top)
 var(--n-padding-left)
 var(--n-padding-bottom)
 var(--n-padding-left);
 `,[u(`main`,`
 font-weight: var(--n-title-font-weight);
 transition: color .3s var(--n-bezier);
 flex: 1;
 min-width: 0;
 color: var(--n-title-text-color);
 `),u(`extra`,`
 display: flex;
 align-items: center;
 font-size: var(--n-font-size);
 font-weight: 400;
 transition: color .3s var(--n-bezier);
 color: var(--n-text-color);
 `),u(`close`,`
 margin: 0 0 0 8px;
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 `)]),u(`action`,`
 box-sizing: border-box;
 transition:
 background-color .3s var(--n-bezier),
 border-color .3s var(--n-bezier);
 background-clip: padding-box;
 background-color: var(--n-action-color);
 `),E,o(`card-content`,[p(`&:first-child`,`
 padding-top: var(--n-padding-bottom);
 `)]),u(`content-scrollbar`,`
 display: flex;
 flex-direction: column;
 `,[p(`>`,[o(`scrollbar-container`,[p(`>`,[E])])]),p(`&:first-child >`,[o(`scrollbar-container`,[p(`>`,[o(`card-content`,`
 padding-top: var(--n-padding-bottom);
 `)])])])]),u(`footer`,`
 box-sizing: border-box;
 padding: 0 var(--n-padding-left) var(--n-padding-bottom) var(--n-padding-left);
 font-size: var(--n-font-size);
 `,[p(`&:first-child`,`
 padding-top: var(--n-padding-bottom);
 `)]),u(`action`,`
 background-color: var(--n-action-color);
 padding: var(--n-padding-bottom) var(--n-padding-left);
 border-bottom-left-radius: var(--n-border-radius);
 border-bottom-right-radius: var(--n-border-radius);
 `)]),o(`card-cover`,`
 overflow: hidden;
 width: 100%;
 border-radius: var(--n-border-radius) var(--n-border-radius) 0 0;
 `,[p(`img`,`
 display: block;
 width: 100%;
 `)]),l(`bordered`,`
 border: 1px solid var(--n-border-color);
 `,[p(`&:target`,`border-color: var(--n-color-target);`)]),l(`action-segmented`,[p(`>`,[u(`action`,[p(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)])])]),l(`content-segmented, content-soft-segmented`,[p(`>`,[o(`card-content`,`
 transition: border-color 0.3s var(--n-bezier);
 `,[p(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)]),u(`content-scrollbar`,`
 transition: border-color 0.3s var(--n-bezier);
 `,[p(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)])])]),l(`footer-segmented, footer-soft-segmented`,[p(`>`,[u(`footer`,`
 transition: border-color 0.3s var(--n-bezier);
 `,[p(`&:not(:first-child)`,`
 border-top: 1px solid var(--n-border-color);
 `)])])]),l(`embedded`,`
 background-color: var(--n-color-embedded);
 `)]),a(o(`card`,`
 background: var(--n-color-modal);
 `,[l(`embedded`,`
 background-color: var(--n-color-embedded-modal);
 `)])),m(o(`card`,`
 background: var(--n-color-popover);
 `,[l(`embedded`,`
 background-color: var(--n-color-embedded-popover);
 `)]))]),O={title:[String,Function],contentClass:String,contentStyle:[Object,String],contentScrollable:Boolean,headerClass:String,headerStyle:[Object,String],headerExtraClass:String,headerExtraStyle:[Object,String],footerClass:String,footerStyle:[Object,String],embedded:Boolean,segmented:{type:[Boolean,Object],default:!1},size:String,bordered:{type:Boolean,default:!0},closable:Boolean,hoverable:Boolean,role:String,onClose:[Function,Array],tag:{type:String,default:`div`},cover:Function,content:[String,Function],footer:Function,action:Function,headerExtra:Function,closeFocusable:Boolean},k=S(O),A=Object.assign(Object.assign({},d.props),O),j=n({name:`Card`,props:A,slots:Object,setup(e){let n=()=>{let{onClose:t}=e;t&&_(t)},{inlineThemeDisabled:a,mergedClsPrefixRef:o,mergedRtlRef:c,mergedComponentPropsRef:l}=i(e),u=d(`Card`,`-card`,D,T,e,o),f=g(`Card`,c,o),p=t(()=>e.size||l?.value?.Card?.size||`medium`),m=t(()=>{let e=p.value,{self:{color:t,colorModal:n,colorTarget:r,textColor:i,titleTextColor:a,titleFontWeight:o,borderColor:c,actionColor:l,borderRadius:d,lineHeight:f,closeIconColor:m,closeIconColorHover:h,closeIconColorPressed:g,closeColorHover:_,closeColorPressed:v,closeBorderRadius:y,closeIconSize:x,closeSize:S,boxShadow:C,colorPopover:w,colorEmbedded:T,colorEmbeddedModal:E,colorEmbeddedPopover:D,[s(`padding`,e)]:O,[s(`fontSize`,e)]:k,[s(`titleFontSize`,e)]:A},common:{cubicBezierEaseInOut:j}}=u.value,{top:M,left:N,bottom:P}=b(O);return{"--n-bezier":j,"--n-border-radius":d,"--n-color":t,"--n-color-modal":n,"--n-color-popover":w,"--n-color-embedded":T,"--n-color-embedded-modal":E,"--n-color-embedded-popover":D,"--n-color-target":r,"--n-text-color":i,"--n-line-height":f,"--n-action-color":l,"--n-title-text-color":a,"--n-title-font-weight":o,"--n-close-icon-color":m,"--n-close-icon-color-hover":h,"--n-close-icon-color-pressed":g,"--n-close-color-hover":_,"--n-close-color-pressed":v,"--n-border-color":c,"--n-box-shadow":C,"--n-padding-top":M,"--n-padding-bottom":P,"--n-padding-left":N,"--n-font-size":k,"--n-title-font-size":A,"--n-close-size":S,"--n-close-icon-size":x,"--n-close-border-radius":y}}),h=a?r(`card`,t(()=>p.value[0]),m,e):void 0;return{rtlEnabled:f,mergedClsPrefix:o,mergedTheme:u,handleCloseClick:n,cssVars:a?void 0:m,themeClass:h?.themeClass,onRender:h?.onRender}},render(){let{segmented:t,bordered:n,hoverable:r,mergedClsPrefix:i,rtlEnabled:a,onRender:o,embedded:s,tag:c,$slots:l}=this;return o?.(),e(c,{class:[`${i}-card`,this.themeClass,s&&`${i}-card--embedded`,{[`${i}-card--rtl`]:a,[`${i}-card--content-scrollable`]:this.contentScrollable,[`${i}-card--content${typeof t!=`boolean`&&t.content===`soft`?`-soft`:``}-segmented`]:t===!0||t!==!1&&t.content,[`${i}-card--footer${typeof t!=`boolean`&&t.footer===`soft`?`-soft`:``}-segmented`]:t===!0||t!==!1&&t.footer,[`${i}-card--action-segmented`]:t===!0||t!==!1&&t.action,[`${i}-card--bordered`]:n,[`${i}-card--hoverable`]:r}],style:this.cssVars,role:this.role},h(l.cover,t=>{let n=this.cover?v([this.cover()]):t;return n&&e(`div`,{class:`${i}-card-cover`,role:`none`},n)}),h(l.header,t=>{let{title:n}=this,r=n?v(typeof n==`function`?[n()]:[n]):t;return r||this.closable?e(`div`,{class:[`${i}-card-header`,this.headerClass],style:this.headerStyle,role:`heading`},e(`div`,{class:`${i}-card-header__main`,role:`heading`},r),h(l[`header-extra`],t=>{let n=this.headerExtra?v([this.headerExtra()]):t;return n&&e(`div`,{class:[`${i}-card-header__extra`,this.headerExtraClass],style:this.headerExtraStyle},n)}),this.closable&&e(x,{clsPrefix:i,class:`${i}-card-header__close`,onClick:this.handleCloseClick,focusable:this.closeFocusable,absolute:!0})):null}),h(l.default,t=>{let{content:n}=this,r=n?v(typeof n==`function`?[n()]:[n]):t;return r?this.contentScrollable?e(y,{class:`${i}-card__content-scrollbar`,contentClass:[`${i}-card-content`,this.contentClass],contentStyle:this.contentStyle},r):e(`div`,{class:[`${i}-card-content`,this.contentClass],style:this.contentStyle,role:`none`},r):null}),h(l.footer,t=>{let n=this.footer?v([this.footer()]):t;return n&&e(`div`,{class:[`${i}-card__footer`,this.footerClass],style:this.footerStyle,role:`none`},n)}),h(l.action,t=>{let n=this.action?v([this.action()]):t;return n&&e(`div`,{class:`${i}-card__action`,role:`none`},n)}))}});export{T as a,A as i,k as n,w as o,O as r,j as t};