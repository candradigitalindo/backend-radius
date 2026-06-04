import{N as e,S as t,T as n,at as r,i,it as a,k as o,l as s,lt as c,q as l,r as u,w as d,y as f}from"./runtime-core.esm-bundler-jDnCq53A.js";import{I as p,St as m,V as h,a as g,dt as _,i as v,o as y,pt as b,ut as x,xt as S}from"./light-Vq2WEi6e.js";function C(e,t){if(e===void 0)return!1;if(t){let{context:{ids:n}}=t;return n.has(e)}return S(e)!==null}function w(){let t=r(!1);return e(()=>{t.value=!0}),a(t)}function T(e,...t){if(Array.isArray(e))e.forEach(e=>T(e,...t));else return e(...t)}function E(e){return e.some(e=>n(e)?!(e.type===u||e.type===i&&!E(e.children)):!0)?e:null}function D(e,t){return e&&E(e())||t()}function O(e,t,n){return e&&E(e(t))||n(t)}function k(e,t){return t(e&&E(e())||null)}function A(e,t,n){return n(e&&E(e(t))||null)}function j(e){return!(e&&E(e()))}function M(e,t,n){if(!t)return;let r=h(),i=s(()=>{let{value:n}=t;if(!n)return;let r=n[e];if(r)return r}),a=d(p,null),c=()=>{l(()=>{let{value:t}=n,o=`${t}${e}Rtl`;if(C(o,r))return;let{value:s}=i;s&&s.style.mount({id:o,head:!0,anchorMetaName:y,props:{bPrefix:t?`.${t}-`:void 0},ssr:r,parent:a?.styleMountTarget})})};return r?c():o(c),i}function N(e,t,n){if(!t)return;let r=h(),i=d(p,null),a=()=>{let a=n.value;t.mount({id:a===void 0?e:a+e,head:!0,anchorMetaName:y,props:{bPrefix:a?`.${a}-`:void 0},ssr:r,parent:i?.styleMountTarget}),i?.preflightStyleDisabled||v.mount({id:`n-global`,head:!0,anchorMetaName:y,ssr:r,parent:i?.styleMountTarget})};r?a():o(a)}var P=f({name:`BaseIconSwitchTransition`,setup(e,{slots:n}){let r=w();return()=>t(m,{name:`icon-switch-transition`,appear:r.value},n)}}),{cubicBezierEaseInOut:F}=g;function I({originalTransform:e=``,left:t=0,top:n=0,transition:r=`all .3s ${F} !important`}={}){return[x(`&.icon-switch-transition-enter-from, &.icon-switch-transition-leave-to`,{transform:`${e} scale(0.75)`,left:t,top:n,opacity:0}),x(`&.icon-switch-transition-enter-to, &.icon-switch-transition-leave-from`,{transform:`scale(1) ${e}`,left:t,top:n,opacity:1}),x(`&.icon-switch-transition-enter-active, &.icon-switch-transition-leave-active`,{transformOrigin:`center`,position:`absolute`,left:t,top:n,transition:r})]}var L=x([x(`@keyframes rotator`,`
 0% {
 -webkit-transform: rotate(0deg);
 transform: rotate(0deg);
 }
 100% {
 -webkit-transform: rotate(360deg);
 transform: rotate(360deg);
 }`),_(`base-loading`,`
 position: relative;
 line-height: 0;
 width: 1em;
 height: 1em;
 `,[b(`transition-wrapper`,`
 position: absolute;
 width: 100%;
 height: 100%;
 `,[I()]),b(`placeholder`,`
 position: absolute;
 left: 50%;
 top: 50%;
 transform: translateX(-50%) translateY(-50%);
 `,[I({left:`50%`,top:`50%`,originalTransform:`translateX(-50%) translateY(-50%)`})]),b(`container`,`
 animation: rotator 3s linear infinite both;
 `,[b(`icon`,`
 height: 1em;
 width: 1em;
 `)])])]),R=`1.6s`,z={strokeWidth:{type:Number,default:28},stroke:{type:String,default:void 0},scale:{type:Number,default:1},radius:{type:Number,default:100}},B=f({name:`BaseLoading`,props:Object.assign({clsPrefix:{type:String,required:!0},show:{type:Boolean,default:!0}},z),setup(e){N(`-base-loading`,L,c(e,`clsPrefix`))},render(){let{clsPrefix:e,radius:n,strokeWidth:r,stroke:i,scale:a}=this,o=n/a;return t(`div`,{class:`${e}-base-loading`,role:`img`,"aria-label":`loading`},t(P,null,{default:()=>this.show?t(`div`,{key:`icon`,class:`${e}-base-loading__transition-wrapper`},t(`div`,{class:`${e}-base-loading__container`},t(`svg`,{class:`${e}-base-loading__icon`,viewBox:`0 0 ${2*o} ${2*o}`,xmlns:`http://www.w3.org/2000/svg`,style:{color:i}},t(`g`,null,t(`animateTransform`,{attributeName:`transform`,type:`rotate`,values:`0 ${o} ${o};270 ${o} ${o}`,begin:`0s`,dur:R,fill:`freeze`,repeatCount:`indefinite`}),t(`circle`,{class:`${e}-base-loading__icon`,fill:`none`,stroke:`currentColor`,"stroke-width":r,"stroke-linecap":`round`,cx:o,cy:o,r:n-r/2,"stroke-dasharray":5.67*n,"stroke-dashoffset":18.48*n},t(`animateTransform`,{attributeName:`transform`,type:`rotate`,values:`0 ${o} ${o};135 ${o} ${o};450 ${o} ${o}`,begin:`0s`,dur:R,fill:`freeze`,repeatCount:`indefinite`}),t(`animate`,{attributeName:`stroke-dashoffset`,values:`${5.67*n};${1.42*n};${5.67*n}`,begin:`0s`,dur:R,fill:`freeze`,repeatCount:`indefinite`})))))):t(`div`,{key:`placeholder`,class:`${e}-base-loading__placeholder`},this.$slots)}))}});export{N as a,j as c,k as d,A as f,P as i,D as l,w as m,z as n,M as o,T as p,I as r,E as s,B as t,O as u};