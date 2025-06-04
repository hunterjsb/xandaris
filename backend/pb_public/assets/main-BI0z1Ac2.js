(function(){const e=document.createElement("link").relList;if(e&&e.supports&&e.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))s(i);new MutationObserver(i=>{for(const n of i)if(n.type==="childList")for(const o of n.addedNodes)o.tagName==="LINK"&&o.rel==="modulepreload"&&s(o)}).observe(document,{childList:!0,subtree:!0});function t(i){const n={};return i.integrity&&(n.integrity=i.integrity),i.referrerPolicy&&(n.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?n.credentials="include":i.crossOrigin==="anonymous"?n.credentials="omit":n.credentials="same-origin",n}function s(i){if(i.ep)return;i.ep=!0;const n=t(i);fetch(i.href,n)}})();class O extends Error{constructor(e){var t,s,i,n;super("ClientResponseError"),this.url="",this.status=0,this.response={},this.isAbort=!1,this.originalError=null,Object.setPrototypeOf(this,O.prototype),e!==null&&typeof e=="object"&&(this.url=typeof e.url=="string"?e.url:"",this.status=typeof e.status=="number"?e.status:0,this.isAbort=!!e.isAbort,this.originalError=e.originalError,e.response!==null&&typeof e.response=="object"?this.response=e.response:e.data!==null&&typeof e.data=="object"?this.response=e.data:this.response={}),this.originalError||e instanceof O||(this.originalError=e),typeof DOMException<"u"&&e instanceof DOMException&&(this.isAbort=!0),this.name="ClientResponseError "+this.status,this.message=(t=this.response)==null?void 0:t.message,this.message||(this.isAbort?this.message="The request was autocancelled. You can find more info in https://github.com/pocketbase/js-sdk#auto-cancellation.":(n=(i=(s=this.originalError)==null?void 0:s.cause)==null?void 0:i.message)!=null&&n.includes("ECONNREFUSED ::1")?this.message="Failed to connect to the PocketBase server. Try changing the SDK URL from localhost to 127.0.0.1 (https://github.com/pocketbase/js-sdk/issues/21).":this.message="Something went wrong while processing your request.")}get data(){return this.response}toJSON(){return{...this}}}const K=/^[\u0009\u0020-\u007e\u0080-\u00ff]+$/;function Pe(m,e){const t={};if(typeof m!="string")return t;const s=Object.assign({},{}).decode||Ee;let i=0;for(;i<m.length;){const n=m.indexOf("=",i);if(n===-1)break;let o=m.indexOf(";",i);if(o===-1)o=m.length;else if(o<n){i=m.lastIndexOf(";",n-1)+1;continue}const a=m.slice(i,n).trim();if(t[a]===void 0){let r=m.slice(n+1,o).trim();r.charCodeAt(0)===34&&(r=r.slice(1,-1));try{t[a]=s(r)}catch{t[a]=r}}i=o+1}return t}function fe(m,e,t){const s=Object.assign({},t||{}),i=s.encode||Fe;if(!K.test(m))throw new TypeError("argument name is invalid");const n=i(e);if(n&&!K.test(n))throw new TypeError("argument val is invalid");let o=m+"="+n;if(s.maxAge!=null){const a=s.maxAge-0;if(isNaN(a)||!isFinite(a))throw new TypeError("option maxAge is invalid");o+="; Max-Age="+Math.floor(a)}if(s.domain){if(!K.test(s.domain))throw new TypeError("option domain is invalid");o+="; Domain="+s.domain}if(s.path){if(!K.test(s.path))throw new TypeError("option path is invalid");o+="; Path="+s.path}if(s.expires){if(!function(r){return Object.prototype.toString.call(r)==="[object Date]"||r instanceof Date}(s.expires)||isNaN(s.expires.valueOf()))throw new TypeError("option expires is invalid");o+="; Expires="+s.expires.toUTCString()}if(s.httpOnly&&(o+="; HttpOnly"),s.secure&&(o+="; Secure"),s.priority)switch(typeof s.priority=="string"?s.priority.toLowerCase():s.priority){case"low":o+="; Priority=Low";break;case"medium":o+="; Priority=Medium";break;case"high":o+="; Priority=High";break;default:throw new TypeError("option priority is invalid")}if(s.sameSite)switch(typeof s.sameSite=="string"?s.sameSite.toLowerCase():s.sameSite){case!0:o+="; SameSite=Strict";break;case"lax":o+="; SameSite=Lax";break;case"strict":o+="; SameSite=Strict";break;case"none":o+="; SameSite=None";break;default:throw new TypeError("option sameSite is invalid")}return o}function Ee(m){return m.indexOf("%")!==-1?decodeURIComponent(m):m}function Fe(m){return encodeURIComponent(m)}const Re=typeof navigator<"u"&&navigator.product==="ReactNative"||typeof global<"u"&&global.HermesInternal;let xe;function H(m){if(m)try{const e=decodeURIComponent(xe(m.split(".")[1]).split("").map(function(t){return"%"+("00"+t.charCodeAt(0).toString(16)).slice(-2)}).join(""));return JSON.parse(e)||{}}catch{}return{}}function we(m,e=0){let t=H(m);return!(Object.keys(t).length>0&&(!t.exp||t.exp-e>Date.now()/1e3))}xe=typeof atob!="function"||Re?m=>{let e=String(m).replace(/=+$/,"");if(e.length%4==1)throw new Error("'atob' failed: The string to be decoded is not correctly encoded.");for(var t,s,i=0,n=0,o="";s=e.charAt(n++);~s&&(t=i%4?64*t+s:s,i++%4)?o+=String.fromCharCode(255&t>>(-2*i&6)):0)s="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=".indexOf(s);return o}:atob;const ge="pb_auth";class Ie{constructor(){this.baseToken="",this.baseModel=null,this._onChangeCallbacks=[]}get token(){return this.baseToken}get model(){return this.baseModel}get isValid(){return!we(this.token)}get isAdmin(){return H(this.token).type==="admin"}get isAuthRecord(){return H(this.token).type==="authRecord"}save(e,t){this.baseToken=e||"",this.baseModel=t||null,this.triggerChange()}clear(){this.baseToken="",this.baseModel=null,this.triggerChange()}loadFromCookie(e,t=ge){const s=Pe(e||"")[t]||"";let i={};try{i=JSON.parse(s),(typeof i===null||typeof i!="object"||Array.isArray(i))&&(i={})}catch{}this.save(i.token||"",i.model||null)}exportToCookie(e,t=ge){var r,l;const s={secure:!0,sameSite:!0,httpOnly:!0,path:"/"},i=H(this.token);s.expires=i!=null&&i.exp?new Date(1e3*i.exp):new Date("1970-01-01"),e=Object.assign({},s,e);const n={token:this.token,model:this.model?JSON.parse(JSON.stringify(this.model)):null};let o=fe(t,JSON.stringify(n),e);const a=typeof Blob<"u"?new Blob([o]).size:o.length;if(n.model&&a>4096){n.model={id:(r=n==null?void 0:n.model)==null?void 0:r.id,email:(l=n==null?void 0:n.model)==null?void 0:l.email};const c=["collectionId","username","verified"];for(const h in this.model)c.includes(h)&&(n.model[h]=this.model[h]);o=fe(t,JSON.stringify(n),e)}return o}onChange(e,t=!1){return this._onChangeCallbacks.push(e),t&&e(this.token,this.model),()=>{for(let s=this._onChangeCallbacks.length-1;s>=0;s--)if(this._onChangeCallbacks[s]==e)return delete this._onChangeCallbacks[s],void this._onChangeCallbacks.splice(s,1)}}triggerChange(){for(const e of this._onChangeCallbacks)e&&e(this.token,this.model)}}class Me extends Ie{constructor(e="pocketbase_auth"){super(),this.storageFallback={},this.storageKey=e,this._bindStorageEvent()}get token(){return(this._storageGet(this.storageKey)||{}).token||""}get model(){return(this._storageGet(this.storageKey)||{}).model||null}save(e,t){this._storageSet(this.storageKey,{token:e,model:t}),super.save(e,t)}clear(){this._storageRemove(this.storageKey),super.clear()}_storageGet(e){if(typeof window<"u"&&(window!=null&&window.localStorage)){const t=window.localStorage.getItem(e)||"";try{return JSON.parse(t)}catch{return t}}return this.storageFallback[e]}_storageSet(e,t){if(typeof window<"u"&&(window!=null&&window.localStorage)){let s=t;typeof t!="string"&&(s=JSON.stringify(t)),window.localStorage.setItem(e,s)}else this.storageFallback[e]=t}_storageRemove(e){var t;typeof window<"u"&&(window!=null&&window.localStorage)&&((t=window.localStorage)==null||t.removeItem(e)),delete this.storageFallback[e]}_bindStorageEvent(){typeof window<"u"&&(window!=null&&window.localStorage)&&window.addEventListener&&window.addEventListener("storage",e=>{if(e.key!=this.storageKey)return;const t=this._storageGet(this.storageKey)||{};super.save(t.token||"",t.model||null)})}}class q{constructor(e){this.client=e}}class Oe extends q{async getAll(e){return e=Object.assign({method:"GET"},e),this.client.send("/api/settings",e)}async update(e,t){return t=Object.assign({method:"PATCH",body:e},t),this.client.send("/api/settings",t)}async testS3(e="storage",t){return t=Object.assign({method:"POST",body:{filesystem:e}},t),this.client.send("/api/settings/test/s3",t).then(()=>!0)}async testEmail(e,t,s){return s=Object.assign({method:"POST",body:{email:e,template:t}},s),this.client.send("/api/settings/test/email",s).then(()=>!0)}async generateAppleClientSecret(e,t,s,i,n,o){return o=Object.assign({method:"POST",body:{clientId:e,teamId:t,keyId:s,privateKey:i,duration:n}},o),this.client.send("/api/settings/apple/generate-client-secret",o)}}class Q extends q{decode(e){return e}async getFullList(e,t){if(typeof e=="number")return this._getFullList(e,t);let s=500;return(t=Object.assign({},e,t)).batch&&(s=t.batch,delete t.batch),this._getFullList(s,t)}async getList(e=1,t=30,s){return(s=Object.assign({method:"GET"},s)).query=Object.assign({page:e,perPage:t},s.query),this.client.send(this.baseCrudPath,s).then(i=>{var n;return i.items=((n=i.items)==null?void 0:n.map(o=>this.decode(o)))||[],i})}async getFirstListItem(e,t){return(t=Object.assign({requestKey:"one_by_filter_"+this.baseCrudPath+"_"+e},t)).query=Object.assign({filter:e,skipTotal:1},t.query),this.getList(1,1,t).then(s=>{var i;if(!((i=s==null?void 0:s.items)!=null&&i.length))throw new O({status:404,response:{code:404,message:"The requested resource wasn't found.",data:{}}});return s.items[0]})}async getOne(e,t){if(!e)throw new O({url:this.client.buildUrl(this.baseCrudPath+"/"),status:404,response:{code:404,message:"Missing required record id.",data:{}}});return t=Object.assign({method:"GET"},t),this.client.send(this.baseCrudPath+"/"+encodeURIComponent(e),t).then(s=>this.decode(s))}async create(e,t){return t=Object.assign({method:"POST",body:e},t),this.client.send(this.baseCrudPath,t).then(s=>this.decode(s))}async update(e,t,s){return s=Object.assign({method:"PATCH",body:t},s),this.client.send(this.baseCrudPath+"/"+encodeURIComponent(e),s).then(i=>this.decode(i))}async delete(e,t){return t=Object.assign({method:"DELETE"},t),this.client.send(this.baseCrudPath+"/"+encodeURIComponent(e),t).then(()=>!0)}_getFullList(e=500,t){(t=t||{}).query=Object.assign({skipTotal:1},t.query);let s=[],i=async n=>this.getList(n,e||500,t).then(o=>{const a=o.items;return s=s.concat(a),a.length==o.perPage?i(n+1):s});return i(1)}}function M(m,e,t,s){const i=s!==void 0;return i||t!==void 0?i?(console.warn(m),e.body=Object.assign({},e.body,t),e.query=Object.assign({},e.query,s),e):Object.assign(e,t):e}function Z(m){var e;(e=m._resetAutoRefresh)==null||e.call(m)}class Le extends Q{get baseCrudPath(){return"/api/admins"}async update(e,t,s){return super.update(e,t,s).then(i=>{var n,o;return((n=this.client.authStore.model)==null?void 0:n.id)===i.id&&((o=this.client.authStore.model)==null?void 0:o.collectionId)===void 0&&this.client.authStore.save(this.client.authStore.token,i),i})}async delete(e,t){return super.delete(e,t).then(s=>{var i,n;return s&&((i=this.client.authStore.model)==null?void 0:i.id)===e&&((n=this.client.authStore.model)==null?void 0:n.collectionId)===void 0&&this.client.authStore.clear(),s})}authResponse(e){const t=this.decode((e==null?void 0:e.admin)||{});return e!=null&&e.token&&(e!=null&&e.admin)&&this.client.authStore.save(e.token,t),Object.assign({},e,{token:(e==null?void 0:e.token)||"",admin:t})}async authWithPassword(e,t,s,i){let n={method:"POST",body:{identity:e,password:t}};n=M("This form of authWithPassword(email, pass, body?, query?) is deprecated. Consider replacing it with authWithPassword(email, pass, options?).",n,s,i);const o=n.autoRefreshThreshold;delete n.autoRefreshThreshold,n.autoRefresh||Z(this.client);let a=await this.client.send(this.baseCrudPath+"/auth-with-password",n);return a=this.authResponse(a),o&&function(l,c,h,d){Z(l);const u=l.beforeSend,p=l.authStore.model,f=l.authStore.onChange((v,g)=>{(!v||(g==null?void 0:g.id)!=(p==null?void 0:p.id)||(g!=null&&g.collectionId||p!=null&&p.collectionId)&&(g==null?void 0:g.collectionId)!=(p==null?void 0:p.collectionId))&&Z(l)});l._resetAutoRefresh=function(){f(),l.beforeSend=u,delete l._resetAutoRefresh},l.beforeSend=async(v,g)=>{var C;const y=l.authStore.token;if((C=g.query)!=null&&C.autoRefresh)return u?u(v,g):{url:v,sendOptions:g};let x=l.authStore.isValid;if(x&&we(l.authStore.token,c))try{await h()}catch{x=!1}x||await d();const S=g.headers||{};for(let k in S)if(k.toLowerCase()=="authorization"&&y==S[k]&&l.authStore.token){S[k]=l.authStore.token;break}return g.headers=S,u?u(v,g):{url:v,sendOptions:g}}}(this.client,o,()=>this.authRefresh({autoRefresh:!0}),()=>this.authWithPassword(e,t,Object.assign({autoRefresh:!0},n))),a}async authRefresh(e,t){let s={method:"POST"};return s=M("This form of authRefresh(body?, query?) is deprecated. Consider replacing it with authRefresh(options?).",s,e,t),this.client.send(this.baseCrudPath+"/auth-refresh",s).then(this.authResponse.bind(this))}async requestPasswordReset(e,t,s){let i={method:"POST",body:{email:e}};return i=M("This form of requestPasswordReset(email, body?, query?) is deprecated. Consider replacing it with requestPasswordReset(email, options?).",i,t,s),this.client.send(this.baseCrudPath+"/request-password-reset",i).then(()=>!0)}async confirmPasswordReset(e,t,s,i,n){let o={method:"POST",body:{token:e,password:t,passwordConfirm:s}};return o=M("This form of confirmPasswordReset(resetToken, password, passwordConfirm, body?, query?) is deprecated. Consider replacing it with confirmPasswordReset(resetToken, password, passwordConfirm, options?).",o,i,n),this.client.send(this.baseCrudPath+"/confirm-password-reset",o).then(()=>!0)}}const Ae=["requestKey","$cancelKey","$autoCancel","fetch","headers","body","query","params","cache","credentials","headers","integrity","keepalive","method","mode","redirect","referrer","referrerPolicy","signal","window"];function Se(m){if(m){m.query=m.query||{};for(let e in m)Ae.includes(e)||(m.query[e]=m[e],delete m[e])}}class Ce extends q{constructor(){super(...arguments),this.clientId="",this.eventSource=null,this.subscriptions={},this.lastSentSubscriptions=[],this.maxConnectTimeout=15e3,this.reconnectAttempts=0,this.maxReconnectAttempts=1/0,this.predefinedReconnectIntervals=[200,300,500,1e3,1200,1500,2e3],this.pendingConnects=[]}get isConnected(){return!!this.eventSource&&!!this.clientId&&!this.pendingConnects.length}async subscribe(e,t,s){var o;if(!e)throw new Error("topic must be set.");let i=e;if(s){Se(s=Object.assign({},s));const a="options="+encodeURIComponent(JSON.stringify({query:s.query,headers:s.headers}));i+=(i.includes("?")?"&":"?")+a}const n=function(a){const r=a;let l;try{l=JSON.parse(r==null?void 0:r.data)}catch{}t(l||{})};return this.subscriptions[i]||(this.subscriptions[i]=[]),this.subscriptions[i].push(n),this.isConnected?this.subscriptions[i].length===1?await this.submitSubscriptions():(o=this.eventSource)==null||o.addEventListener(i,n):await this.connect(),async()=>this.unsubscribeByTopicAndListener(e,n)}async unsubscribe(e){var s;let t=!1;if(e){const i=this.getSubscriptionsByTopic(e);for(let n in i)if(this.hasSubscriptionListeners(n)){for(let o of this.subscriptions[n])(s=this.eventSource)==null||s.removeEventListener(n,o);delete this.subscriptions[n],t||(t=!0)}}else this.subscriptions={};this.hasSubscriptionListeners()?t&&await this.submitSubscriptions():this.disconnect()}async unsubscribeByPrefix(e){var s;let t=!1;for(let i in this.subscriptions)if((i+"?").startsWith(e)){t=!0;for(let n of this.subscriptions[i])(s=this.eventSource)==null||s.removeEventListener(i,n);delete this.subscriptions[i]}t&&(this.hasSubscriptionListeners()?await this.submitSubscriptions():this.disconnect())}async unsubscribeByTopicAndListener(e,t){var n;let s=!1;const i=this.getSubscriptionsByTopic(e);for(let o in i){if(!Array.isArray(this.subscriptions[o])||!this.subscriptions[o].length)continue;let a=!1;for(let r=this.subscriptions[o].length-1;r>=0;r--)this.subscriptions[o][r]===t&&(a=!0,delete this.subscriptions[o][r],this.subscriptions[o].splice(r,1),(n=this.eventSource)==null||n.removeEventListener(o,t));a&&(this.subscriptions[o].length||delete this.subscriptions[o],s||this.hasSubscriptionListeners(o)||(s=!0))}this.hasSubscriptionListeners()?s&&await this.submitSubscriptions():this.disconnect()}hasSubscriptionListeners(e){var t,s;if(this.subscriptions=this.subscriptions||{},e)return!!((t=this.subscriptions[e])!=null&&t.length);for(let i in this.subscriptions)if((s=this.subscriptions[i])!=null&&s.length)return!0;return!1}async submitSubscriptions(){if(this.clientId)return this.addAllSubscriptionListeners(),this.lastSentSubscriptions=this.getNonEmptySubscriptionKeys(),this.client.send("/api/realtime",{method:"POST",body:{clientId:this.clientId,subscriptions:this.lastSentSubscriptions},requestKey:this.getSubscriptionsCancelKey()}).catch(e=>{if(!(e!=null&&e.isAbort))throw e})}getSubscriptionsCancelKey(){return"realtime_"+this.clientId}getSubscriptionsByTopic(e){const t={};e=e.includes("?")?e:e+"?";for(let s in this.subscriptions)(s+"?").startsWith(e)&&(t[s]=this.subscriptions[s]);return t}getNonEmptySubscriptionKeys(){const e=[];for(let t in this.subscriptions)this.subscriptions[t].length&&e.push(t);return e}addAllSubscriptionListeners(){if(this.eventSource){this.removeAllSubscriptionListeners();for(let e in this.subscriptions)for(let t of this.subscriptions[e])this.eventSource.addEventListener(e,t)}}removeAllSubscriptionListeners(){if(this.eventSource)for(let e in this.subscriptions)for(let t of this.subscriptions[e])this.eventSource.removeEventListener(e,t)}async connect(){if(!(this.reconnectAttempts>0))return new Promise((e,t)=>{this.pendingConnects.push({resolve:e,reject:t}),this.pendingConnects.length>1||this.initConnect()})}initConnect(){this.disconnect(!0),clearTimeout(this.connectTimeoutId),this.connectTimeoutId=setTimeout(()=>{this.connectErrorHandler(new Error("EventSource connect took too long."))},this.maxConnectTimeout),this.eventSource=new EventSource(this.client.buildUrl("/api/realtime")),this.eventSource.onerror=e=>{this.connectErrorHandler(new Error("Failed to establish realtime connection."))},this.eventSource.addEventListener("PB_CONNECT",e=>{const t=e;this.clientId=t==null?void 0:t.lastEventId,this.submitSubscriptions().then(async()=>{let s=3;for(;this.hasUnsentSubscriptions()&&s>0;)s--,await this.submitSubscriptions()}).then(()=>{for(let i of this.pendingConnects)i.resolve();this.pendingConnects=[],this.reconnectAttempts=0,clearTimeout(this.reconnectTimeoutId),clearTimeout(this.connectTimeoutId);const s=this.getSubscriptionsByTopic("PB_CONNECT");for(let i in s)for(let n of s[i])n(e)}).catch(s=>{this.clientId="",this.connectErrorHandler(s)})})}hasUnsentSubscriptions(){const e=this.getNonEmptySubscriptionKeys();if(e.length!=this.lastSentSubscriptions.length)return!0;for(const t of e)if(!this.lastSentSubscriptions.includes(t))return!0;return!1}connectErrorHandler(e){if(clearTimeout(this.connectTimeoutId),clearTimeout(this.reconnectTimeoutId),!this.clientId&&!this.reconnectAttempts||this.reconnectAttempts>this.maxReconnectAttempts){for(let s of this.pendingConnects)s.reject(new O(e));return this.pendingConnects=[],void this.disconnect()}this.disconnect(!0);const t=this.predefinedReconnectIntervals[this.reconnectAttempts]||this.predefinedReconnectIntervals[this.predefinedReconnectIntervals.length-1];this.reconnectAttempts++,this.reconnectTimeoutId=setTimeout(()=>{this.initConnect()},t)}disconnect(e=!1){var t;if(clearTimeout(this.connectTimeoutId),clearTimeout(this.reconnectTimeoutId),this.removeAllSubscriptionListeners(),this.client.cancelRequest(this.getSubscriptionsCancelKey()),(t=this.eventSource)==null||t.close(),this.eventSource=null,this.clientId="",!e){this.reconnectAttempts=0;for(let s of this.pendingConnects)s.resolve();this.pendingConnects=[]}}}class De extends Q{constructor(e,t){super(e),this.collectionIdOrName=t}get baseCrudPath(){return this.baseCollectionPath+"/records"}get baseCollectionPath(){return"/api/collections/"+encodeURIComponent(this.collectionIdOrName)}async subscribe(e,t,s){if(!e)throw new Error("Missing topic.");if(!t)throw new Error("Missing subscription callback.");return this.client.realtime.subscribe(this.collectionIdOrName+"/"+e,t,s)}async unsubscribe(e){return e?this.client.realtime.unsubscribe(this.collectionIdOrName+"/"+e):this.client.realtime.unsubscribeByPrefix(this.collectionIdOrName)}async getFullList(e,t){if(typeof e=="number")return super.getFullList(e,t);const s=Object.assign({},e,t);return super.getFullList(s)}async getList(e=1,t=30,s){return super.getList(e,t,s)}async getFirstListItem(e,t){return super.getFirstListItem(e,t)}async getOne(e,t){return super.getOne(e,t)}async create(e,t){return super.create(e,t)}async update(e,t,s){return super.update(e,t,s).then(i=>{var n,o,a;return((n=this.client.authStore.model)==null?void 0:n.id)!==(i==null?void 0:i.id)||((o=this.client.authStore.model)==null?void 0:o.collectionId)!==this.collectionIdOrName&&((a=this.client.authStore.model)==null?void 0:a.collectionName)!==this.collectionIdOrName||this.client.authStore.save(this.client.authStore.token,i),i})}async delete(e,t){return super.delete(e,t).then(s=>{var i,n,o;return!s||((i=this.client.authStore.model)==null?void 0:i.id)!==e||((n=this.client.authStore.model)==null?void 0:n.collectionId)!==this.collectionIdOrName&&((o=this.client.authStore.model)==null?void 0:o.collectionName)!==this.collectionIdOrName||this.client.authStore.clear(),s})}authResponse(e){const t=this.decode((e==null?void 0:e.record)||{});return this.client.authStore.save(e==null?void 0:e.token,t),Object.assign({},e,{token:(e==null?void 0:e.token)||"",record:t})}async listAuthMethods(e){return e=Object.assign({method:"GET"},e),this.client.send(this.baseCollectionPath+"/auth-methods",e).then(t=>Object.assign({},t,{usernamePassword:!!(t!=null&&t.usernamePassword),emailPassword:!!(t!=null&&t.emailPassword),authProviders:Array.isArray(t==null?void 0:t.authProviders)?t==null?void 0:t.authProviders:[]}))}async authWithPassword(e,t,s,i){let n={method:"POST",body:{identity:e,password:t}};return n=M("This form of authWithPassword(usernameOrEmail, pass, body?, query?) is deprecated. Consider replacing it with authWithPassword(usernameOrEmail, pass, options?).",n,s,i),this.client.send(this.baseCollectionPath+"/auth-with-password",n).then(o=>this.authResponse(o))}async authWithOAuth2Code(e,t,s,i,n,o,a){let r={method:"POST",body:{provider:e,code:t,codeVerifier:s,redirectUrl:i,createData:n}};return r=M("This form of authWithOAuth2Code(provider, code, codeVerifier, redirectUrl, createData?, body?, query?) is deprecated. Consider replacing it with authWithOAuth2Code(provider, code, codeVerifier, redirectUrl, createData?, options?).",r,o,a),this.client.send(this.baseCollectionPath+"/auth-with-oauth2",r).then(l=>this.authResponse(l))}authWithOAuth2(...e){if(e.length>1||typeof(e==null?void 0:e[0])=="string")return console.warn("PocketBase: This form of authWithOAuth2() is deprecated and may get removed in the future. Please replace with authWithOAuth2Code() OR use the authWithOAuth2() realtime form as shown in https://pocketbase.io/docs/authentication/#oauth2-integration."),this.authWithOAuth2Code((e==null?void 0:e[0])||"",(e==null?void 0:e[1])||"",(e==null?void 0:e[2])||"",(e==null?void 0:e[3])||"",(e==null?void 0:e[4])||{},(e==null?void 0:e[5])||{},(e==null?void 0:e[6])||{});const t=(e==null?void 0:e[0])||{};let s=null;t.urlCallback||(s=ye(void 0));const i=new Ce(this.client);function n(){s==null||s.close(),i.unsubscribe()}const o={},a=t.requestKey;return a&&(o.requestKey=a),this.listAuthMethods(o).then(r=>{var d;const l=r.authProviders.find(u=>u.name===t.provider);if(!l)throw new O(new Error(`Missing or invalid provider "${t.provider}".`));const c=this.client.buildUrl("/api/oauth2-redirect"),h=a?(d=this.client.cancelControllers)==null?void 0:d[a]:void 0;return h&&(h.signal.onabort=()=>{n()}),new Promise(async(u,p)=>{var f;try{await i.subscribe("@oauth2",async x=>{var C;const S=i.clientId;try{if(!x.state||S!==x.state)throw new Error("State parameters don't match.");if(x.error||!x.code)throw new Error("OAuth2 redirect error or missing code: "+x.error);const k=Object.assign({},t);delete k.provider,delete k.scopes,delete k.createData,delete k.urlCallback,(C=h==null?void 0:h.signal)!=null&&C.onabort&&(h.signal.onabort=null);const P=await this.authWithOAuth2Code(l.name,x.code,l.codeVerifier,c,t.createData,k);u(P)}catch(k){p(new O(k))}n()});const v={state:i.clientId};(f=t.scopes)!=null&&f.length&&(v.scope=t.scopes.join(" "));const g=this._replaceQueryParams(l.authUrl+c,v);await(t.urlCallback||function(x){s?s.location.href=x:s=ye(x)})(g)}catch(v){n(),p(new O(v))}})}).catch(r=>{throw n(),r})}async authRefresh(e,t){let s={method:"POST"};return s=M("This form of authRefresh(body?, query?) is deprecated. Consider replacing it with authRefresh(options?).",s,e,t),this.client.send(this.baseCollectionPath+"/auth-refresh",s).then(i=>this.authResponse(i))}async requestPasswordReset(e,t,s){let i={method:"POST",body:{email:e}};return i=M("This form of requestPasswordReset(email, body?, query?) is deprecated. Consider replacing it with requestPasswordReset(email, options?).",i,t,s),this.client.send(this.baseCollectionPath+"/request-password-reset",i).then(()=>!0)}async confirmPasswordReset(e,t,s,i,n){let o={method:"POST",body:{token:e,password:t,passwordConfirm:s}};return o=M("This form of confirmPasswordReset(token, password, passwordConfirm, body?, query?) is deprecated. Consider replacing it with confirmPasswordReset(token, password, passwordConfirm, options?).",o,i,n),this.client.send(this.baseCollectionPath+"/confirm-password-reset",o).then(()=>!0)}async requestVerification(e,t,s){let i={method:"POST",body:{email:e}};return i=M("This form of requestVerification(email, body?, query?) is deprecated. Consider replacing it with requestVerification(email, options?).",i,t,s),this.client.send(this.baseCollectionPath+"/request-verification",i).then(()=>!0)}async confirmVerification(e,t,s){let i={method:"POST",body:{token:e}};return i=M("This form of confirmVerification(token, body?, query?) is deprecated. Consider replacing it with confirmVerification(token, options?).",i,t,s),this.client.send(this.baseCollectionPath+"/confirm-verification",i).then(()=>{const n=H(e),o=this.client.authStore.model;return o&&!o.verified&&o.id===n.id&&o.collectionId===n.collectionId&&(o.verified=!0,this.client.authStore.save(this.client.authStore.token,o)),!0})}async requestEmailChange(e,t,s){let i={method:"POST",body:{newEmail:e}};return i=M("This form of requestEmailChange(newEmail, body?, query?) is deprecated. Consider replacing it with requestEmailChange(newEmail, options?).",i,t,s),this.client.send(this.baseCollectionPath+"/request-email-change",i).then(()=>!0)}async confirmEmailChange(e,t,s,i){let n={method:"POST",body:{token:e,password:t}};return n=M("This form of confirmEmailChange(token, password, body?, query?) is deprecated. Consider replacing it with confirmEmailChange(token, password, options?).",n,s,i),this.client.send(this.baseCollectionPath+"/confirm-email-change",n).then(()=>{const o=H(e),a=this.client.authStore.model;return a&&a.id===o.id&&a.collectionId===o.collectionId&&this.client.authStore.clear(),!0})}async listExternalAuths(e,t){return t=Object.assign({method:"GET"},t),this.client.send(this.baseCrudPath+"/"+encodeURIComponent(e)+"/external-auths",t)}async unlinkExternalAuth(e,t,s){return s=Object.assign({method:"DELETE"},s),this.client.send(this.baseCrudPath+"/"+encodeURIComponent(e)+"/external-auths/"+encodeURIComponent(t),s).then(()=>!0)}_replaceQueryParams(e,t={}){let s=e,i="";e.indexOf("?")>=0&&(s=e.substring(0,e.indexOf("?")),i=e.substring(e.indexOf("?")+1));const n={},o=i.split("&");for(const a of o){if(a=="")continue;const r=a.split("=");n[decodeURIComponent(r[0].replace(/\+/g," "))]=decodeURIComponent((r[1]||"").replace(/\+/g," "))}for(let a in t)t.hasOwnProperty(a)&&(t[a]==null?delete n[a]:n[a]=t[a]);i="";for(let a in n)n.hasOwnProperty(a)&&(i!=""&&(i+="&"),i+=encodeURIComponent(a.replace(/%20/g,"+"))+"="+encodeURIComponent(n[a].replace(/%20/g,"+")));return i!=""?s+"?"+i:s}}function ye(m){if(typeof window>"u"||!(window!=null&&window.open))throw new O(new Error("Not in a browser context - please pass a custom urlCallback function."));let e=1024,t=768,s=window.innerWidth,i=window.innerHeight;e=e>s?s:e,t=t>i?i:t;let n=s/2-e/2,o=i/2-t/2;return window.open(m,"popup_window","width="+e+",height="+t+",top="+o+",left="+n+",resizable,menubar=no")}class Ue extends Q{get baseCrudPath(){return"/api/collections"}async import(e,t=!1,s){return s=Object.assign({method:"PUT",body:{collections:e,deleteMissing:t}},s),this.client.send(this.baseCrudPath+"/import",s).then(()=>!0)}}class Ne extends q{async getList(e=1,t=30,s){return(s=Object.assign({method:"GET"},s)).query=Object.assign({page:e,perPage:t},s.query),this.client.send("/api/logs",s)}async getOne(e,t){if(!e)throw new O({url:this.client.buildUrl("/api/logs/"),status:404,response:{code:404,message:"Missing required log id.",data:{}}});return t=Object.assign({method:"GET"},t),this.client.send("/api/logs/"+encodeURIComponent(e),t)}async getStats(e){return e=Object.assign({method:"GET"},e),this.client.send("/api/logs/stats",e)}}class je extends q{async check(e){return e=Object.assign({method:"GET"},e),this.client.send("/api/health",e)}}class Be extends q{getUrl(e,t,s={}){if(!t||!(e!=null&&e.id)||!(e!=null&&e.collectionId)&&!(e!=null&&e.collectionName))return"";const i=[];i.push("api"),i.push("files"),i.push(encodeURIComponent(e.collectionId||e.collectionName)),i.push(encodeURIComponent(e.id)),i.push(encodeURIComponent(t));let n=this.client.buildUrl(i.join("/"));if(Object.keys(s).length){s.download===!1&&delete s.download;const o=new URLSearchParams(s);n+=(n.includes("?")?"&":"?")+o}return n}async getToken(e){return e=Object.assign({method:"POST"},e),this.client.send("/api/files/token",e).then(t=>(t==null?void 0:t.token)||"")}}class ze extends q{async getFullList(e){return e=Object.assign({method:"GET"},e),this.client.send("/api/backups",e)}async create(e,t){return t=Object.assign({method:"POST",body:{name:e}},t),this.client.send("/api/backups",t).then(()=>!0)}async upload(e,t){return t=Object.assign({method:"POST",body:e},t),this.client.send("/api/backups/upload",t).then(()=>!0)}async delete(e,t){return t=Object.assign({method:"DELETE"},t),this.client.send(`/api/backups/${encodeURIComponent(e)}`,t).then(()=>!0)}async restore(e,t){return t=Object.assign({method:"POST"},t),this.client.send(`/api/backups/${encodeURIComponent(e)}/restore`,t).then(()=>!0)}getDownloadUrl(e,t){return this.client.buildUrl(`/api/backups/${encodeURIComponent(t)}?token=${encodeURIComponent(e)}`)}}class qe{constructor(e="/",t,s="en-US"){this.cancelControllers={},this.recordServices={},this.enableAutoCancellation=!0,this.baseUrl=e,this.lang=s,this.authStore=t||new Me,this.admins=new Le(this),this.collections=new Ue(this),this.files=new Be(this),this.logs=new Ne(this),this.settings=new Oe(this),this.realtime=new Ce(this),this.health=new je(this),this.backups=new ze(this)}collection(e){return this.recordServices[e]||(this.recordServices[e]=new De(this,e)),this.recordServices[e]}autoCancellation(e){return this.enableAutoCancellation=!!e,this}cancelRequest(e){return this.cancelControllers[e]&&(this.cancelControllers[e].abort(),delete this.cancelControllers[e]),this}cancelAllRequests(){for(let e in this.cancelControllers)this.cancelControllers[e].abort();return this.cancelControllers={},this}filter(e,t){if(!t)return e;for(let s in t){let i=t[s];switch(typeof i){case"boolean":case"number":i=""+i;break;case"string":i="'"+i.replace(/'/g,"\\'")+"'";break;default:i=i===null?"null":i instanceof Date?"'"+i.toISOString().replace("T"," ")+"'":"'"+JSON.stringify(i).replace(/'/g,"\\'")+"'"}e=e.replaceAll("{:"+s+"}",i)}return e}getFileUrl(e,t,s={}){return this.files.getUrl(e,t,s)}buildUrl(e){var s;let t=this.baseUrl;return typeof window>"u"||!window.location||t.startsWith("https://")||t.startsWith("http://")||(t=(s=window.location.origin)!=null&&s.endsWith("/")?window.location.origin.substring(0,window.location.origin.length-1):window.location.origin||"",this.baseUrl.startsWith("/")||(t+=window.location.pathname||"/",t+=t.endsWith("/")?"":"/"),t+=this.baseUrl),e&&(t+=t.endsWith("/")?"":"/",t+=e.startsWith("/")?e.substring(1):e),t}async send(e,t){t=this.initSendOptions(e,t);let s=this.buildUrl(e);if(this.beforeSend){const i=Object.assign({},await this.beforeSend(s,t));i.url!==void 0||i.options!==void 0?(s=i.url||s,t=i.options||t):Object.keys(i).length&&(t=i,console!=null&&console.warn&&console.warn("Deprecated format of beforeSend return: please use `return { url, options }`, instead of `return options`."))}if(t.query!==void 0){const i=this.serializeQueryParams(t.query);i&&(s+=(s.includes("?")?"&":"?")+i),delete t.query}return this.getHeader(t.headers,"Content-Type")=="application/json"&&t.body&&typeof t.body!="string"&&(t.body=JSON.stringify(t.body)),(t.fetch||fetch)(s,t).then(async i=>{let n={};try{n=await i.json()}catch{}if(this.afterSend&&(n=await this.afterSend(i,n)),i.status>=400)throw new O({url:i.url,status:i.status,data:n});return n}).catch(i=>{throw new O(i)})}initSendOptions(e,t){if((t=Object.assign({method:"GET"},t)).body=this.convertToFormDataIfNeeded(t.body),Se(t),t.query=Object.assign({},t.params,t.query),t.requestKey===void 0&&(t.$autoCancel===!1||t.query.$autoCancel===!1?t.requestKey=null:(t.$cancelKey||t.query.$cancelKey)&&(t.requestKey=t.$cancelKey||t.query.$cancelKey)),delete t.$autoCancel,delete t.query.$autoCancel,delete t.$cancelKey,delete t.query.$cancelKey,this.getHeader(t.headers,"Content-Type")!==null||this.isFormData(t.body)||(t.headers=Object.assign({},t.headers,{"Content-Type":"application/json"})),this.getHeader(t.headers,"Accept-Language")===null&&(t.headers=Object.assign({},t.headers,{"Accept-Language":this.lang})),this.authStore.token&&this.getHeader(t.headers,"Authorization")===null&&(t.headers=Object.assign({},t.headers,{Authorization:this.authStore.token})),this.enableAutoCancellation&&t.requestKey!==null){const s=t.requestKey||(t.method||"GET")+e;delete t.requestKey,this.cancelRequest(s);const i=new AbortController;this.cancelControllers[s]=i,t.signal=i.signal}return t}convertToFormDataIfNeeded(e){if(typeof FormData>"u"||e===void 0||typeof e!="object"||e===null||this.isFormData(e)||!this.hasBlobField(e))return e;const t=new FormData;for(const s in e){const i=e[s];if(typeof i!="object"||this.hasBlobField({data:i})){const n=Array.isArray(i)?i:[i];for(let o of n)t.append(s,o)}else{let n={};n[s]=i,t.append("@jsonPayload",JSON.stringify(n))}}return t}hasBlobField(e){for(const t in e){const s=Array.isArray(e[t])?e[t]:[e[t]];for(const i of s)if(typeof Blob<"u"&&i instanceof Blob||typeof File<"u"&&i instanceof File)return!0}return!1}getHeader(e,t){e=e||{},t=t.toLowerCase();for(let s in e)if(s.toLowerCase()==t)return e[s];return null}isFormData(e){return e&&(e.constructor.name==="FormData"||typeof FormData<"u"&&e instanceof FormData)}serializeQueryParams(e){const t=[];for(const s in e){if(e[s]===null)continue;const i=e[s],n=encodeURIComponent(s);if(Array.isArray(i))for(const o of i)t.push(n+"="+encodeURIComponent(o));else i instanceof Date?t.push(n+"="+encodeURIComponent(i.toISOString())):typeof i!==null&&typeof i=="object"?t.push(n+"="+encodeURIComponent(JSON.stringify(i))):t.push(n+"="+encodeURIComponent(i))}return t.join("&")}}const Te="http://localhost:8090",b=new qe(Te);function D(m){var e;if(!((e=m.message)!=null&&e.includes("autocancelled")||m.status===0))throw m}class $e{constructor(){this.callbacks=[],this.user=null,this.checkAuthStatus(),b.authStore.onChange(()=>{this.checkAuthStatus(),this.notifyCallbacks()})}checkAuthStatus(){this.user=b.authStore.isValid?b.authStore.model:null}subscribe(e){this.callbacks.push(e),e(this.user)}unsubscribe(e){this.callbacks=this.callbacks.filter(t=>t!==e)}notifyCallbacks(){this.callbacks.forEach(e=>e(this.user))}async loginWithDiscord(){try{return await b.collection("users").authWithOAuth2({provider:"discord"})}catch(e){throw console.error("Discord login failed:",e),e}}logout(){b.authStore.clear()}isLoggedIn(){return b.authStore.isValid}getUser(){return this.user}}const $=new $e;class ke{constructor(){this.ws=null,this.callbacks={systems:[],fleets:[],trades:[],tick:[],fleet_orders:[]}}subscribe(e,t){this.callbacks[e]&&this.callbacks[e].push(t)}unsubscribe(e,t){this.callbacks[e]&&(this.callbacks[e]=this.callbacks[e].filter(s=>s!==t))}async getHyperlanes(){if(!b.authStore.isValid)throw new Error("Not authenticated");try{return await b.send("/api/hyperlanes",{method:"GET"})}catch(e){throw console.error("Failed to fetch hyperlanes:",e),e}}notifyCallbacks(e,t){this.callbacks[e]&&this.callbacks[e].forEach(s=>s(t))}connectWebSocket(){try{this.ws=new WebSocket(`${Te.replace("http","ws")}/api/stream`),this.ws.onopen=()=>{console.log("WebSocket connected"),this.updateConnectionStatus("connected"),b.authStore.isValid&&setTimeout(()=>{this.ws&&this.ws.readyState===WebSocket.OPEN&&this.ws.send(JSON.stringify({type:"auth",token:b.authStore.token}))},100)},this.ws.onmessage=e=>{try{const t=JSON.parse(e.data);this.handleWebSocketMessage(t)}catch(t){console.error("Failed to parse WebSocket message:",t)}},this.ws.onclose=()=>{console.log("WebSocket disconnected"),this.updateConnectionStatus("disconnected"),setTimeout(()=>this.connectWebSocket(),5e3)},this.ws.onerror=e=>{console.error("WebSocket error:",e),this.updateConnectionStatus("error")}}catch(e){console.error("Failed to connect WebSocket:",e),this.updateConnectionStatus("error")}}updateConnectionStatus(e){const t=document.getElementById("ws-status");t&&(t.textContent=e==="connected"?"🟢":e==="error"?"🔴":"🟡",t.title=`WebSocket: ${e}`)}handleWebSocketMessage(e){switch(e.type){case"tick":this.notifyCallbacks("tick",e.payload);break;case"system_update":this.notifyCallbacks("systems",e.payload);break;case"fleet_update":this.notifyCallbacks("fleets",e.payload);break;case"trade_update":this.notifyCallbacks("trades",e.payload);break;case"fleet_order_update":this.notifyCallbacks("fleet_orders",e.payload);break;default:console.log("Unknown WebSocket message type:",e.type)}}async getFleetOrders(e=null){if(b.authStore.isValid,!e&&b.authStore.isValid&&(e=b.authStore.model.id),!e)return console.warn("getFleetOrders called without userId and no authenticated user."),[];try{const s=['(status != "completed" && status != "failed" && status != "cancelled")',`user_id = "${e}"`].join(" && ");return await b.collection("fleet_orders").getFullList({filter:s,sort:"execute_at_tick",requestKey:`getFleetOrders-${e}-${Date.now()}`})}catch(t){try{D(t)}catch(s){console.error("Failed to fetch fleet orders:",s)}return[]}}async getSystems(){try{return await b.collection("systems").getFullList({sort:"x,y"})}catch(e){return console.error("Failed to fetch systems:",e),[]}}async getPlayer(e){try{return await b.collection("users").getOne(e,{requestKey:`getPlayer-${e}-${Date.now()}`})}catch(t){return console.error("Failed to fetch player details:",t),null}}async getPlayerCredits(e){try{return(await b.collection("players").getOne(e)).credits}catch(t){return console.error("Failed to fetch player credits:",t),0}}async getUserResources(){var e;try{return(await b.send("/api/user/resources",{method:"GET",requestKey:`getUserResources-${Date.now()}`})).resources}catch(t){return t.status===0&&((e=t.message)!=null&&e.includes("autocancelled"))?(console.debug("User resources request was auto-cancelled (expected behavior)"),null):(console.error("Failed to fetch user resources:",t),{credits:0,food:0,ore:0,fuel:0,metal:0,oil:0,titanium:0,xanium:0})}}async getSystem(e){try{return await b.collection("systems").getOne(e)}catch(t){return console.error("Failed to fetch system:",t),null}}async getFleets(e=null){try{const t=e?`/api/fleets?owner_id=${e}`:"/api/fleets",s=await fetch(`${b.baseUrl}${t}`,{headers:{Authorization:b.authStore.token||""}});if(!s.ok)throw new Error(`HTTP ${s.status}`);return(await s.json()).items||[]}catch(t){try{D(t)}catch(s){console.error("Failed to fetch fleets:",s)}return[]}}async getTrades(e=null){try{const t=e?`owner_id = "${e}"`:"";return await b.collection("trade_routes").getFullList({filter:t,sort:"created"})}catch(t){try{D(t)}catch(s){console.error("Failed to fetch trades:",s)}return[]}}async getBuildings(e=null){try{let t="/api/buildings";const s={};return e&&(s.owner_id=e),(await b.send(t,{method:"GET",params:s})).items||[]}catch(t){try{D(t)}catch(s){console.error("Failed to fetch buildings:",s)}return[]}}async getTreaties(e=null){try{return[]}catch(t){try{D(t)}catch(s){console.error("Failed to fetch treaties:",s)}return[]}}async sendFleet(e,t,s,i=null){if(!b.authStore.isValid)throw new Error("Not authenticated");try{const n={from_id:e,to_id:t,strength:s};return i&&(n.fleet_id=i),await b.send("/api/orders/fleet",{method:"POST",body:JSON.stringify(n),headers:{"Content-Type":"application/json"}})}catch(n){throw console.error("Failed to send fleet:",n),n}}async sendFleetRoute(e,t){if(!b.authStore.isValid)throw new Error("Not authenticated");try{return await b.send("/api/orders/fleet-route",{method:"POST",body:JSON.stringify({fleet_id:e,route_path:t}),headers:{"Content-Type":"application/json"}})}catch(s){throw console.error("Failed to send fleet route:",s),s}}async queueBuilding(e,t,s){if(!b.authStore.isValid)throw new Error("Not authenticated");try{return await b.send("/api/orders/build",{method:"POST",body:JSON.stringify({planet_id:e,building_type:t,fleet_id:s}),headers:{"Content-Type":"application/json"}})}catch(i){throw console.error("Failed to queue building:",i),i}}async getShipCargo(e){if(!b.authStore.isValid)throw new Error("Not authenticated");try{return await b.send(`/api/ship_cargo?fleet_id=${e}`,{method:"GET",headers:{"Content-Type":"application/json"}})}catch(t){throw console.error("Failed to get ship cargo:",t),t}}async createTradeRoute(e,t,s,i){if(!b.authStore.isValid)throw new Error("Not authenticated");try{return await b.send("/api/orders/trade",{method:"POST",body:JSON.stringify({from_id:e,to_id:t,cargo:s,capacity:i}),headers:{"Content-Type":"application/json"}})}catch(n){throw console.error("Failed to create trade route:",n),n}}async proposeTreaty(e,t,s){if(!b.authStore.isValid)throw new Error("Not authenticated");try{return await b.send("/diplomacy",{method:"POST",body:JSON.stringify({player_id:e,type:t,terms:s}),headers:{"Content-Type":"application/json"}})}catch(i){throw console.error("Failed to propose treaty:",i),i}}async getMap(){var e;try{return await b.send("/api/map",{method:"GET"})}catch(t){return(e=t.message)!=null&&e.includes("autocancelled")||console.error("Failed to fetch map:",t),null}}async getStatus(){try{return await b.send("/api/status",{method:"GET"})}catch(e){try{D(e)}catch(t){console.error("Failed to fetch status:",t)}return null}}async getBuildingTypes(){try{return(await b.send("/api/building_types",{method:"GET"})).items||[]}catch(e){try{D(e)}catch(t){console.error("Failed to fetch building types:",t)}return[]}}async getResourceTypes(){try{return(await b.send("/api/resource_types",{method:"GET"})).items||[]}catch(e){try{D(e)}catch(t){console.error("Failed to fetch resource types:",t)}return[]}}async getPopulations(e=null){try{let t="/api/collections/populations/records";const s={};return e&&(s.filter=`owner_id='${e}'`),s.expand="employed_at,planet_id",(await b.send(t,{method:"GET",params:s})).items||[]}catch(t){try{D(t)}catch(s){console.error("Failed to fetch populations:",s)}return[]}}}const w=new ke;$.subscribe(m=>{w.ws||w.connectWebSocket()});w.connectWebSocket();const Ve=Object.freeze(Object.defineProperty({__proto__:null,AuthManager:$e,GameDataManager:ke,authManager:$,gameData:w,pb:b},Symbol.toStringTag,{value:"Module"}));class _e{constructor(){this.systems=[],this.fleets=[],this.trades=[],this.treaties=[],this.buildings=[],this.populations=[],this.fleetOrders=[],this.hyperlanes=[],this.mapData=null,this.selectedSystem=null,this.selectedSystemPlanets=[],this.currentTick=1,this.ticksPerMinute=6,this.buildingTypes=[],this.resourceTypes=[],this.playerResources={credits:0,food:0,ore:0,goods:0,fuel:0},this.creditIncome=0,this.shipCargo=new Map,this.callbacks=[],this.initialized=!1,this.updatingResources=!1,this.updateTimer=null,this.tickRefreshTimer=null,this.pendingUpdate=!1,this.isUpdating=!1,$.subscribe(e=>{e&&!this.initialized?this.initialize():e||this.reset()}),this.loadMapData(),w.subscribe("systems",e=>this.updateSystems(e)),w.subscribe("fleets",e=>this.updateFleets(e)),w.subscribe("trades",e=>this.updateTrades(e)),w.subscribe("tick",e=>this.handleTick(e)),w.subscribe("fleet_orders",e=>this.updateFleetOrders(e))}async initialize(){if(!this.initialized)try{await this.loadGameData(),this.initialized=!0}catch(e){console.error("Failed to initialize game state:",e)}}async loadGameData(){var e;try{const t=(e=$.getUser())==null?void 0:e.id,s=await w.getMap();s&&s.systems&&(this.systems=s.systems,this.mapData=s);const i=await w.getHyperlanes();if(i&&(this.hyperlanes=i),t&&(this.fleets=await w.getFleets(t),await new Promise(o=>setTimeout(o,50)),this.trades=await w.getTrades(t),await new Promise(o=>setTimeout(o,50)),this.treaties=await w.getTreaties(t),await new Promise(o=>setTimeout(o,50)),this.buildings=await w.getBuildings(t),await new Promise(o=>setTimeout(o,50)),this.populations=await w.getPopulations(t),await new Promise(o=>setTimeout(o,50)),this.fleetOrders=await w.getFleetOrders(t),await new Promise(o=>setTimeout(o,50)),this.fleets&&this.fleets.length>0&&!this.cargoLoaded)){for(const o of this.fleets)try{const a=await this.getShipCargo(o.id);await new Promise(r=>setTimeout(r,25))}catch(a){console.warn(`Failed to load cargo for fleet ${o.id}:`,a)}this.cargoLoaded=!0}const n=await w.getStatus();n&&(this.currentTick=n.current_tick||1,this.ticksPerMinute=n.ticks_per_minute||6);try{const o=await w.getBuildingTypes();o&&(this.buildingTypes=o)}catch(o){console.warn("Failed to load building types:",o),this.buildingTypes=[]}try{const o=await w.getResourceTypes();o&&(this.resourceTypes=o)}catch(o){console.warn("Failed to load resource types:",o),this.resourceTypes=[]}if(this.updatePlayerResources(),t&&this.fleets&&this.fleets.length>0&&this.systems&&this.systems.length>0){const o=this.fleets[0];o&&o.current_system&&(this.centerOnFleetSystem=o.current_system)}this.notifyCallbacks()}catch(t){console.error("Failed to load game data:",t)}}async refreshGameData(){if(!this.initialized){await this.initialize();return}await this.loadGameData()}async lightweightTickUpdate(){if($.getUser())try{await this.updatePlayerResources();const t=await w.getStatus();t&&(this.currentTick=t.current_tick||this.currentTick,this.ticksPerMinute=t.ticks_per_minute||this.ticksPerMinute),this.notifyCallbacks()}catch(t){console.warn("Failed to perform lightweight tick update:",t)}}handleTick(e){console.log("Received tick:",e),this.currentTick=e.tick||e.current_tick||this.currentTick,this.tickRefreshTimer&&clearTimeout(this.tickRefreshTimer),this.tickRefreshTimer=setTimeout(()=>{this.currentTick%10===0?this.refreshGameData():this.lightweightTickUpdate()},500)}reset(){this.systems=[],this.fleets=[],this.trades=[],this.treaties=[],this.buildings=[],this.populations=[],this.fleetOrders=[],this.hyperlanes=[],this.mapData=null,this.selectedSystem=null,this.selectedSystemPlanets=[],this.currentTick=1,this.ticksPerMinute=6,this.buildingTypes=[],this.resourceTypes=[],this.playerResources={credits:0,food:0,ore:0,goods:0,fuel:0},this.creditIncome=0,this.initialized=!1,this.notifyCallbacks()}async loadMapData(){try{const e=await w.getMap();if(e&&e.systems){this.systems=e.systems,this.mapData=e;const t=await w.getHyperlanes();t&&(this.hyperlanes=t),this.notifyCallbacks()}}catch(e){console.error("Failed to load map data:",e)}}subscribe(e){this.callbacks.push(e),e(this)}unsubscribe(e){this.callbacks=this.callbacks.filter(t=>t!==e)}notifyCallbacks(){if(!this.isUpdating){if(this.updateTimer){this.pendingUpdate=!0;return}this.updateTimer=setTimeout(()=>{this.updateTimer=null,this.isUpdating=!0;try{this.callbacks.forEach(e=>e(this))}finally{this.isUpdating=!1}this.pendingUpdate&&(this.pendingUpdate=!1,this.notifyCallbacks())},16)}}updateSystems(e){if(Array.isArray(e))this.systems=e;else{const t=this.systems.findIndex(s=>s.id===e.id);t>=0?this.systems[t]=e:this.systems.push(e)}this.updatePlayerResources(),this.notifyCallbacks()}updateFleets(e){if(console.log("DEBUG: updateFleets called with:",Array.isArray(e)?`array of ${e.length} fleets`:"single fleet",e),Array.isArray(e)){const t=new Map(this.fleets.map(s=>[s.id,s]));console.log(`DEBUG: Checking ${e.length} fleets for arrivals against ${t.size} old fleets`);for(const s of e){const i=t.get(s.id);i&&(console.log(`DEBUG: Fleet ${s.id} - old dest: "${i.destination_system}", new dest: "${s.destination_system}"`),i.destination_system&&!s.destination_system&&(console.log(`DEBUG: Fleet arrival detected for fleet ${s.id}, old destination: ${i.destination_system}, new destination: ${s.destination_system}`),this.handleFleetArrival(s.id)))}this.fleets=e}else{const t=this.fleets.findIndex(s=>s.id===e.id);if(t>=0){const s=this.fleets[t];this.fleets[t]=e,s.destination_system&&!e.destination_system&&(console.log(`DEBUG: Fleet arrival detected for fleet ${e.id}, old destination: ${s.destination_system}, new destination: ${e.destination_system}`),this.handleFleetArrival(e.id))}else this.fleets.push(e)}this.notifyCallbacks()}handleFleetArrival(e){console.log(`DEBUG: Fleet ${e} arrived, checking for multi-hop continuation`),window.app&&typeof window.app.onFleetArrival=="function"?(console.log(`DEBUG: Calling window.app.onFleetArrival for fleet ${e}`),window.app.onFleetArrival(e)):console.warn(`DEBUG: window.app.onFleetArrival not available for fleet ${e}`)}updateTrades(e){if(Array.isArray(e))this.trades=e;else{const t=this.trades.findIndex(s=>s.id===e.id);t>=0?this.trades[t]=e:this.trades.push(e)}this.notifyCallbacks()}updateFleetOrders(e){if(Array.isArray(e))this.fleetOrders=e;else{const t=this.fleetOrders.findIndex(s=>s.id===e.id);t>=0?this.fleetOrders[t]=e:(this.fleetOrders.push(e),this.fleetOrders.sort((s,i)=>s.execute_at_tick-i.execute_at_tick))}this.notifyCallbacks()}async updatePlayerResources(){var t;const e=$.getUser();if(!e){this.playerResources={credits:0,food:0,ore:0,goods:0,fuel:0},this.creditIncome=0;return}if(!this.updatingResources){this.updatingResources=!0;try{const s=await w.getUserResources();if(s===null)return;if(!this.mapData||!this.mapData.planets){this.playerResources=s,this.creditIncome=0;return}let i=0,n=0,o=0,a=0,r=0;for(const g of this.mapData.planets)if(g.colonized_by===e.id&&(i+=g.Food||0,n+=g.Ore||0,o+=g.Goods||0,a+=g.Fuel||0,g.Buildings))for(const[y,x]of Object.entries(g.Buildings)){const S=this.buildingTypes.find(C=>C.id===y||C.name&&C.name.toLowerCase()===y.toLowerCase());S&&S.name&&S.name.toLowerCase()==="bank"&&(r+=(x||1)*1)}let l=0,c=0,h=0,d=0,u=0,p=0;for(const[g,y]of this.shipCargo.entries())y&&y.cargo&&(l+=y.cargo.ore||0,c+=y.cargo.food||0,h+=y.cargo.fuel||0,d+=y.cargo.metal||0,u+=y.cargo.titanium||0,p+=y.cargo.xanium||0);let f=0,v=0;if(this.buildings&&Array.isArray(this.buildings))for(const g of this.buildings){const y=this.buildingTypes.find(x=>x.id===g.building_type);if(y){const x=(t=y.name)==null?void 0:t.toLowerCase();x==="mine"&&g.res1_stored>0?f+=g.res1_stored:x==="crypto_server"&&g.res1_stored>0&&(v+=g.res1_stored)}}this.playerResources={credits:s.credits,food:i+c,ore:n+l+f,fuel:a+h,metal:d,titanium:u,xanium:p},this.creditIncome=r}finally{this.updatingResources=!1}}}getSystemPlanets(e){return!this.mapData||!this.mapData.planets?[]:this.mapData.planets.filter(s=>s.system_id===e)}selectSystem(e){this.selectedSystem&&this.selectedSystem.id===e||(this.selectedSystem=this.systems.find(t=>t.id===e)||null,this.selectedSystem?this.selectedSystemPlanets=this.getSystemPlanets(this.selectedSystem.id):this.selectedSystemPlanets=[],this.notifyCallbacks())}getSelectedSystem(){return this.selectedSystem}getOwnedSystems(){const e=$.getUser();return e?this.systems.filter(t=>t.owner_id===e.id):[]}getPlayerFleets(){const e=$.getUser();return e?this.fleets.filter(t=>t.owner_id===e.id):[]}getPlayerTrades(){const e=$.getUser();return e?this.trades.filter(t=>t.owner_id===e.id):[]}async sendFleet(e,t,s){return await w.sendFleet(e,t,s)}async queueBuilding(e,t,s){return await w.queueBuilding(e,t,s)}async getShipCargo(e){try{const t=await w.getShipCargo(e);return this.shipCargo.set(e,t),this.notifyCallbacks(),t}catch(t){throw console.error("Failed to load ship cargo:",t),t}}async refreshAllShipCargo(){if(!this.fleets||this.fleets.length===0)return;const e=$.getUser();if(!e)return;const t=this.fleets.filter(s=>s.owner_id===e.id);for(const s of t)try{await this.getShipCargo(s.id),await new Promise(i=>setTimeout(i,25))}catch(i){console.warn(`Failed to refresh cargo for fleet ${s.id}:`,i)}}getFleetCargo(e){return this.shipCargo.get(e)||{cargo:{},used_capacity:0,total_capacity:0}}async createTradeRoute(e,t,s,i){return await w.createTradeRoute(e,t,s,i)}async proposeTreaty(e,t,s){return await w.proposeTreaty(e,t,s)}getPlayerBuildings(){var t;const e=$.getUser();return e?((t=this.buildings)==null?void 0:t.filter(s=>s.owner_id===e.id))||[]:[]}getPlayerBuildingsByType(e){return this.getPlayerBuildings().filter(t=>t.type===e)}}const T=new _e,He=Object.freeze(Object.defineProperty({__proto__:null,GameState:_e,gameState:T},Symbol.toStringTag,{value:"Module"}));class We{constructor(e){this.canvas=document.getElementById(e),this.ctx=this.canvas.getContext("2d"),this.systems=[],this.lanes=[],this.hyperlanes=[],this.fleets=[],this.selectedSystem=null,this.selectedFleet=null,this.hoveredSystem=null,this.hoveredTradeRoutes=[],this.trades=[],this.currentUserId=null,this.cachedTerritorialContours=null,this.territorialCacheKey=null,this.connectedSystems=new Map,this.fleetRoutes=[],this.viewX=0,this.viewY=0,this.zoom=.15,this.maxZoom=2,this.minZoom=.05,this.targetViewX=0,this.targetViewY=0,this.cameraSpeed=.15,this.colors={background:"#000508",starUnowned:"#4080ff",starPlayerOwned:"#00ff66",starOtherOwned:"#f1a9ff",starEnemy:"#ff6b6b",lane:"rgba(64, 128, 255, 0.2)",laneActive:"rgba(241, 169, 255, 0.6)",fleet:"#8b5cf6",selection:"#f1a9ff",grid:"rgba(255, 255, 255, 0.02)",nebula:"rgba(139, 92, 246, 0.1)",starGlow:"rgba(64, 128, 255, 0.3)"},this.animationFrame=null,this.lastTime=0,this.isDirty=!0,this.isMoving=!1,this.setupCanvas(),this.setupEventListeners(),this.startRenderLoop(),this.initialViewSet=!1,this.systemPlanetCounts=new Map}setupCanvas(){this.resizeCanvas(),window.addEventListener("resize",()=>this.resizeCanvas())}resizeCanvas(){const e=this.canvas.getBoundingClientRect();this.canvas.width=e.width,this.canvas.height=e.height}setupEventListeners(){let e=!1,t=0,s=0;this.canvas.addEventListener("mousedown",i=>{if(i.button===0){const n=this.screenToWorld(i.offsetX,i.offsetY),o=this.getFleetAt(n.x,n.y);if(o){this.selectFleet(o,i.offsetX,i.offsetY);return}const a=this.getSystemAt(n.x,n.y);if(a){if(i.shiftKey&&this.selectedFleet){this.canvas.dispatchEvent(new CustomEvent("fleetMoveRequested",{detail:{fromFleet:this.selectedFleet,toSystem:a,shiftKey:!0},bubbles:!0}));return}this.selectSystem(a),this.selectFleet(null,null,null);const r={x:this.canvas.width/2+30,y:this.canvas.height/2-20},l=window.gameState.getSystemPlanets(a.id);this.canvas.dispatchEvent(new CustomEvent("systemSelected",{detail:{system:a,planets:l,screenX:r.x,screenY:r.y},bubbles:!0}))}else this.selectSystem(null),this.canvas.dispatchEvent(new CustomEvent("mapClickedEmpty",{bubbles:!0})),e=!0,t=i.offsetX,s=i.offsetY,this.canvas.style.cursor="grabbing"}}),this.canvas.addEventListener("mousemove",i=>{if(e){const n=i.offsetX-t,o=i.offsetY-s;this.viewX+=n/this.zoom,this.viewY+=o/this.zoom,this.targetViewX=this.viewX,this.targetViewY=this.viewY,this.isDirty=!0,t=i.offsetX,s=i.offsetY}else{const n=this.screenToWorld(i.offsetX,i.offsetY),o=this.getSystemAt(n.x,n.y);o!==this.hoveredSystem&&(this.hoveredSystem=o,this.isDirty=!0),this.showTooltip(this.hoveredSystem,i.offsetX,i.offsetY),this.hoveredSystem&&this.trades&&this.trades.length>0?this.hoveredTradeRoutes=this.trades.filter(a=>a.from_id===this.hoveredSystem.id||a.to_id===this.hoveredSystem.id):this.hoveredTradeRoutes=[]}}),this.canvas.addEventListener("mouseup",i=>{i.button===0&&(e=!1,this.canvas.style.cursor="crosshair")}),this.canvas.addEventListener("contextmenu",i=>{i.preventDefault();const n=this.screenToWorld(i.offsetX,i.offsetY),o=this.getSystemAt(n.x,n.y);o&&this.showContextMenu(o,i.offsetX,i.offsetY)}),this.canvas.addEventListener("wheel",i=>{i.preventDefault();const n=i.deltaY>0?.9:1.1,o=Math.max(this.minZoom,Math.min(this.maxZoom,this.zoom*n));if(o!==this.zoom){const a=this.canvas.getBoundingClientRect(),r=i.clientX-a.left,l=i.clientY-a.top,c=this.screenToWorld(r,l);this.zoom=o;const h=this.screenToWorld(r,l);this.viewX+=c.x-h.x,this.viewY+=c.y-h.y,this.targetViewX=this.viewX,this.targetViewY=this.viewY,this.isDirty=!0}})}screenToWorld(e,t){return{x:(e-this.canvas.width/2)/this.zoom-this.viewX,y:(t-this.canvas.height/2)/this.zoom-this.viewY}}worldToScreen(e,t){return{x:(e+this.viewX)*this.zoom+this.canvas.width/2,y:(t+this.viewY)*this.zoom+this.canvas.height/2}}getSystemAt(e,t){return this.systems.find(i=>{const n=i.x-e,o=i.y-t;return Math.sqrt(n*n+o*o)<=20})}selectSystem(e){if(this.selectedSystem&&e&&this.selectedSystem.id===e.id){this.centerOnSystem(e.id);return}this.selectedSystem=e,e&&this.centerOnSystem(e.id)}updateConnectedSystems(){if(this.connectedSystems.clear(),!this.selectedSystem)return;const e=this.selectedSystem.x,t=this.selectedSystem.y;this.systems.forEach(s=>{if(s.id===this.selectedSystem.id)return;const i=s.x-e,n=s.y-t,o=Math.sqrt(i*i+n*n);if(o>800)return;const a=Math.atan2(n,i)*180/Math.PI;let r;a>=-45&&a<=45?r="right":a>=45&&a<=135?r="down":a>=135||a<=-135?r="left":r="up",(!this.connectedSystems.has(r)||o<this.connectedSystems.get(r).distance)&&this.connectedSystems.set(r,{system:s,distance:o,direction:r})})}showTooltip(e,t,s){const i=document.getElementById("tooltip");if(e&&window.gameState){const n=window.gameState.getSystemPlanets(e.id),o=n.reduce((l,c)=>l+(c.Pop||0),0),r=n.some(l=>l.colonized_by===this.currentUserId)?"You":"Uncolonized";i.innerHTML=`
        <div class="font-semibold">${e.name||`System ${e.id.slice(-4)}`}</div>
        <div class="text-xs">
          <div>Position: ${e.x}, ${e.y}</div>
          <div>Population: ${o.toLocaleString()}</div>
          <div>Owner: ${r}</div>
          <div>Planets: ${n.length}</div>
        </div>
      `,i.style.left=`${t+10}px`,i.style.top=`${s-10}px`,i.classList.remove("hidden")}else i.classList.add("hidden")}showContextMenu(e,t,s){const i=document.getElementById("context-menu");i.style.left=`${t}px`,i.style.top=`${s}px`,i.classList.remove("hidden"),i.dataset.systemId=e.id;const n=o=>{i.contains(o.target)||(i.classList.add("hidden"),document.removeEventListener("click",n))};setTimeout(()=>document.addEventListener("click",n),0)}startRenderLoop(){const e=t=>{const s=t-this.lastTime;this.lastTime=t,this.updateCamera(s),(this.isDirty||this.isMoving)&&(this.clear(),this.drawBackground(),this.drawLanes(),this.drawFleetRoutes(),this.drawCachedTerritorialBorders(),this.drawSystems(),this.drawFleets(s),this.drawUI(),this.isDirty=!1),this.animationFrame=requestAnimationFrame(e)};this.animationFrame=requestAnimationFrame(e)}updateCamera(e){const t=this.cameraSpeed*(e/16),s=this.targetViewX-this.viewX,i=this.targetViewY-this.viewY;Math.abs(s)>.1||Math.abs(i)>.1?(this.viewX+=s*t,this.viewY+=i*t,this.isMoving=!0,this.isDirty=!0):(this.viewX=this.targetViewX,this.viewY=this.targetViewY,this.isMoving=!1)}clear(){this.ctx.fillStyle=this.colors.background,this.ctx.fillRect(0,0,this.canvas.width,this.canvas.height)}drawBackground(){this.ctx.globalAlpha=.1;const e=Date.now()*1e-4;for(let a=0;a<3;a++){const r=a*150,l=Math.sin(e+a)*200+r,c=Math.cos(e*.7+a)*150+r,h=300+Math.sin(e*.5+a)*50,d=this.ctx.createRadialGradient(l,c,0,l,c,h);d.addColorStop(0,this.colors.nebula),d.addColorStop(.5,"rgba(64, 128, 255, 0.05)"),d.addColorStop(1,"rgba(0, 0, 0, 0)"),this.ctx.fillStyle=d,this.ctx.fillRect(0,0,this.canvas.width,this.canvas.height)}this.ctx.globalAlpha=1,this.ctx.strokeStyle=this.colors.grid,this.ctx.lineWidth=1,this.ctx.globalAlpha=.15;const t=100,s=Math.floor((-this.viewX-this.canvas.width/2/this.zoom)/t)*t,i=Math.ceil((-this.viewX+this.canvas.width/2/this.zoom)/t)*t,n=Math.floor((-this.viewY-this.canvas.height/2/this.zoom)/t)*t,o=Math.ceil((-this.viewY+this.canvas.height/2/this.zoom)/t)*t;this.ctx.beginPath();for(let a=s;a<=i;a+=t){const r=this.worldToScreen(a,0);this.ctx.moveTo(r.x,0),this.ctx.lineTo(r.x,this.canvas.height)}for(let a=n;a<=o;a+=t){const r=this.worldToScreen(0,a);this.ctx.moveTo(0,r.y),this.ctx.lineTo(this.canvas.width,r.y)}this.ctx.stroke(),this.ctx.globalAlpha=1}drawLanes(){if(this.drawHyperlanes(),this.drawNavigationLanes(),!this.trades||this.trades.length===0){this.lanes&&this.lanes.length>0&&(this.ctx.strokeStyle=this.colors.lane,this.ctx.lineWidth=2,this.ctx.globalAlpha=.6,this.lanes.forEach(e=>{const t=this.worldToScreen(e.fromX,e.fromY),s=this.worldToScreen(e.toX,e.toY);this.ctx.beginPath(),this.ctx.moveTo(t.x,t.y),this.ctx.lineTo(s.x,s.y),this.ctx.stroke()}),this.ctx.globalAlpha=1);return}this.ctx.globalAlpha=.7,this.trades.forEach(e=>{const t=this.systems.find(a=>a.id===e.from_id),s=this.systems.find(a=>a.id===e.to_id);if(!t||!s)return;const i=this.worldToScreen(t.x,t.y),n=this.worldToScreen(s.x,s.y);let o=this.colors.lane;this.hoveredTradeRoutes&&this.hoveredTradeRoutes.some(a=>a.id===e.id)?(o=this.colors.laneActive,this.ctx.lineWidth=3):this.ctx.lineWidth=2,this.ctx.strokeStyle=o,this.ctx.beginPath(),this.ctx.moveTo(i.x,i.y),this.ctx.lineTo(n.x,n.y),this.ctx.stroke()}),this.ctx.globalAlpha=1}drawHyperlanes(){!this.hyperlanes||this.hyperlanes.length===0||(this.ctx.strokeStyle="rgba(64, 128, 255, 0.12)",this.ctx.lineWidth=1,this.ctx.globalAlpha=.5,this.ctx.setLineDash([]),this.hyperlanes.forEach(e=>{const t=this.systems.find(o=>o.id===e.from_system),s=this.systems.find(o=>o.id===e.to_system);if(!t||!s)return;const i=this.worldToScreen(t.x,t.y),n=this.worldToScreen(s.x,s.y);this.ctx.beginPath(),this.ctx.moveTo(i.x,i.y),this.ctx.lineTo(n.x,n.y),this.ctx.stroke()}),this.ctx.globalAlpha=1)}drawNavigationLanes(){!this.selectedSystem||!this.hyperlanes||this.hyperlanes.length===0||(this.ctx.strokeStyle="rgba(64, 128, 255, 0.4)",this.ctx.lineWidth=2,this.ctx.globalAlpha=.9,this.ctx.setLineDash([4,8]),this.hyperlanes.forEach(e=>{let t=null;if(e.from_system===this.selectedSystem.id?t=this.systems.find(n=>n.id===e.to_system):e.to_system===this.selectedSystem.id&&(t=this.systems.find(n=>n.id===e.from_system)),!t)return;const s=this.worldToScreen(this.selectedSystem.x,this.selectedSystem.y),i=this.worldToScreen(t.x,t.y);this.ctx.beginPath(),this.ctx.moveTo(s.x,s.y),this.ctx.lineTo(i.x,i.y),this.ctx.stroke()}),this.ctx.setLineDash([]),this.ctx.globalAlpha=1)}drawFleetRoutes(){this.fleetRoutes.length!==0&&(this.ctx.strokeStyle="#8b5cf6",this.ctx.lineWidth=3,this.ctx.globalAlpha=.8,this.fleetRoutes.forEach(e=>{const t=this.worldToScreen(e.from.x,e.from.y),s=this.worldToScreen(e.to.x,e.to.y);this.ctx.setLineDash([10,5]),this.ctx.lineDashOffset=-Date.now()/50,this.ctx.beginPath(),this.ctx.moveTo(t.x,t.y),this.ctx.lineTo(s.x,s.y),this.ctx.stroke();const i=Math.atan2(s.y-t.y,s.x-t.x),n=15;this.ctx.setLineDash([]),this.ctx.fillStyle="#8b5cf6",this.ctx.beginPath(),this.ctx.moveTo(s.x,s.y),this.ctx.lineTo(s.x-n*Math.cos(i-Math.PI/6),s.y-n*Math.sin(i-Math.PI/6)),this.ctx.lineTo(s.x-n*Math.cos(i+Math.PI/6),s.y-n*Math.sin(i+Math.PI/6)),this.ctx.closePath(),this.ctx.fill()}),this.ctx.setLineDash([]),this.ctx.globalAlpha=1)}showFleetRoute(e,t){this.fleetRoutes.push({from:e,to:t,timestamp:Date.now()}),setTimeout(()=>{this.fleetRoutes=this.fleetRoutes.filter(s=>s.timestamp!==this.fleetRoutes[this.fleetRoutes.length-1].timestamp),this.isDirty=!0},3e3),this.isDirty=!0}showMultiFleetRoute(e){const t=Date.now();for(let s=0;s<e.length-1;s++)this.fleetRoutes.push({from:e[s],to:e[s+1],timestamp:t,isMultiHop:!0,hopIndex:s});setTimeout(()=>{this.fleetRoutes=this.fleetRoutes.filter(s=>s.timestamp!==t),this.isDirty=!0},5e3),this.isDirty=!0}drawFleetRoutes(){this.fleetRoutes.length!==0&&(this.fleetRoutes.forEach(e=>{const t=this.worldToScreen(e.from.x,e.from.y),s=this.worldToScreen(e.to.x,e.to.y);if(e.isMultiHop){const i=["#8b5cf6","#f59e0b","#ef4444","#10b981","#3b82f6"],n=i[e.hopIndex%i.length];this.ctx.strokeStyle=n,this.ctx.lineWidth=2,this.ctx.globalAlpha=.9,this.ctx.setLineDash([10,5])}else this.ctx.strokeStyle="#8b5cf6",this.ctx.lineWidth=3,this.ctx.globalAlpha=.8,this.ctx.setLineDash([]);this.ctx.beginPath(),this.ctx.moveTo(t.x,t.y),this.ctx.lineTo(s.x,s.y),this.ctx.stroke()}),this.ctx.globalAlpha=1,this.ctx.setLineDash([]))}drawSystems(){this.systems.forEach(e=>{const t=this.worldToScreen(e.x,e.y);if(t.x<-50||t.x>this.canvas.width+50||t.y<-50||t.y>this.canvas.height+50)return;let s=1;this.hoveredSystem&&this.hoveredSystem.id===e.id&&(s=1.2);let i,n=!1,o=!1;for(const[f,v]of this.connectedSystems)if(v.system.id===e.id){n=!0;break}window.gameState&&(o=window.gameState.getSystemPlanets(e.id).some(v=>v.colonized_by===this.currentUserId)),o?i=this.colors.starPlayerOwned:e.owner_id?i=this.colors.starOtherOwned:i=this.colors.starUnowned;const a=this.systemPlanetCounts.get(e.id)||1,r=Math.min(1+(a-1)*.2,2),l=6*this.zoom*r,c=l*s,h=20*this.zoom*s*r,d=this.ctx.createRadialGradient(t.x,t.y,0,t.x,t.y,h);d.addColorStop(0,i+"40"),d.addColorStop(.3,i+"20"),d.addColorStop(1,i+"00"),this.ctx.fillStyle=d,this.ctx.beginPath(),this.ctx.arc(t.x,t.y,h,0,Math.PI*2),this.ctx.fill(),n&&this.selectedSystem&&(this.ctx.strokeStyle="#ffffff80",this.ctx.lineWidth=2,this.ctx.setLineDash([5,5]),this.ctx.beginPath(),this.ctx.arc(t.x,t.y,c+4,0,Math.PI*2),this.ctx.stroke(),this.ctx.setLineDash([]));const u=this.ctx.createRadialGradient(t.x,t.y,0,t.x,t.y,c);if(u.addColorStop(0,"#ffffff"),u.addColorStop(.7,i),u.addColorStop(1,i+"cc"),this.ctx.fillStyle=u,this.ctx.beginPath(),this.ctx.arc(t.x,t.y,c,0,Math.PI*2),this.ctx.fill(),o&&(this.ctx.strokeStyle="#00ff88",this.ctx.lineWidth=2*this.zoom*s,this.ctx.globalAlpha=.9,this.ctx.beginPath(),this.ctx.arc(t.x,t.y,c+2,0,Math.PI*2),this.ctx.stroke(),this.ctx.globalAlpha=1),o&&this.zoom>.6&&(this.ctx.strokeStyle=this.colors.starPlayerOwned,this.ctx.lineWidth=1,this.ctx.globalAlpha=.8,this.ctx.beginPath(),this.ctx.moveTo(t.x-l*1.5,t.y),this.ctx.lineTo(t.x+l*1.5,t.y),this.ctx.moveTo(t.x,t.y-l*1.5),this.ctx.lineTo(t.x,t.y+l*1.5),this.ctx.stroke(),this.ctx.globalAlpha=1),this.selectedSystem&&this.selectedSystem.id===e.id){const f=Date.now()*.005,v=(l+8*this.zoom)*s;this.ctx.strokeStyle=this.colors.selection,this.ctx.lineWidth=3*this.zoom*s,this.ctx.globalAlpha=.9,this.ctx.beginPath(),this.ctx.arc(t.x,t.y,v,0,Math.PI*2),this.ctx.stroke();const g=(l+(6+Math.sin(f)*3)*this.zoom)*s;this.ctx.strokeStyle=this.colors.selection,this.ctx.lineWidth=2*this.zoom*s,this.ctx.globalAlpha=.7+Math.sin(f)*.3,this.ctx.beginPath(),this.ctx.arc(t.x,t.y,g,0,Math.PI*2),this.ctx.stroke(),this.ctx.globalAlpha=1}if(this.zoom>.8){const f=Math.floor(11*this.zoom);this.ctx.font=`${f}px monospace`,this.ctx.textAlign="center";const v=c+5*this.zoom;this.ctx.fillStyle="rgba(0, 0, 0, 0.8)",this.ctx.fillText(e.name||`S${e.id.slice(-3)}`,t.x+1,t.y-v+1),this.ctx.fillStyle="rgba(255, 255, 255, 0.95)",this.ctx.fillText(e.name||`S${e.id.slice(-3)}`,t.x,t.y-v)}let p=0;if(window.gameState&&(p=window.gameState.getSystemPlanets(e.id).reduce((v,g)=>v+(g.Pop||0),0)),p>0&&this.zoom>.5){const f=Math.floor(9*this.zoom);this.ctx.font=`${f}px monospace`,this.ctx.textAlign="center";const v=16*this.zoom,g=12*this.zoom,y=v*s,x=g*s,S=t.y+c+2*this.zoom,C=S+x-3*this.zoom*s;this.ctx.fillStyle="rgba(241, 169, 255, 0.2)",this.ctx.fillRect(t.x-y/2,S,y,x),this.ctx.fillStyle="#f1a9ff",this.ctx.fillText(p.toLocaleString(),t.x,C)}})}drawCachedTerritorialBorders(){if(!window.gameState)return;const e=new Set;this.systems.forEach(n=>{window.gameState.getSystemPlanets(n.id).forEach(a=>{a.colonized_by&&e.add(a.colonized_by)})});const t={[this.currentUserId]:"0, 255, 102"},s=["241, 169, 255","255, 107, 107","139, 92, 246","34, 197, 94","249, 115, 22","236, 72, 153","14, 165, 233","168, 85, 247"];let i=0;e.forEach(n=>{n!==this.currentUserId&&(t[n]=s[i%s.length],i++)}),e.forEach(n=>{var l;const o=this.systems.filter(c=>window.gameState.getSystemPlanets(c.id).some(d=>d.colonized_by===n));if(o.length<1)return;const a=this.createTerritorialCacheKey(o,n);this.playerTerritorialCaches||(this.playerTerritorialCaches={}),((l=this.playerTerritorialCaches[n])==null?void 0:l.cacheKey)!==a&&(this.playerTerritorialCaches[n]={cacheKey:a,contours:this.computeTerritorialContours(o)});const r=this.playerTerritorialCaches[n];if(r!=null&&r.contours){const c=t[n]||"128, 128, 128",h=n===this.currentUserId;this.drawTerritorialContours(r.contours,c,h)}})}createTerritorialCacheKey(e,t){const s=e.map(n=>`${n.id}:${n.x}:${n.y}`).sort().join("|"),i=`${Math.floor(this.viewX/50)}:${Math.floor(this.viewY/50)}:${Math.floor(this.zoom*10)}`;return`${t}:${s}@${i}`}computeTerritorialContours(e){const t=this.getVisibleWorldBounds(),s=40,i=this.calculateInfluenceField(e,t,s);return this.extractTerritorialContours(i,t,s)}drawUnifiedTerritories(e,t="0, 255, 102",s=!0){e.forEach(n=>{this.drawSimpleInfluenceBorder(n,e,200,t,s)})}drawSimpleInfluenceBorder(e,t,s,i,n){this.ctx.save();const o=this.worldToScreen(e.x,e.y),a=s*this.zoom,r=this.ctx.createRadialGradient(o.x,o.y,0,o.x,o.y,a),l=n?.1:.05;r.addColorStop(0,`rgba(${i}, ${l})`),r.addColorStop(1,`rgba(${i}, 0)`),this.ctx.fillStyle=r,this.ctx.beginPath(),this.ctx.arc(o.x,o.y,a,0,Math.PI*2),this.ctx.fill(),this.ctx.restore()}getVisibleWorldBounds(){const t=this.screenToWorld(-200,-200),s=this.screenToWorld(this.canvas.width+200,this.canvas.height+200);return{minX:t.x,minY:t.y,maxX:s.x,maxY:s.y}}calculateInfluenceField(e,t,s){const i=Math.ceil((t.maxX-t.minX)/s),n=Math.ceil((t.maxY-t.minY)/s),o=new Array(n).fill(null).map(()=>new Array(i).fill(0)),a=150;for(let r=0;r<n;r++)for(let l=0;l<i;l++){const c=t.minX+l*s,h=t.minY+r*s;let d=0,u=0;this.systems.filter(f=>{const v=f.x-c,g=f.y-h;return Math.abs(v)<a&&Math.abs(g)<a}).forEach(f=>{const v=f.x-c,g=f.y-h,y=Math.sqrt(v*v+g*g);if(y<a){const x=Math.max(0,1-y/a);e.some(C=>C.id===f.id)?d+=x*x:window.gameState&&window.gameState.getSystemPlanets(f.id).some(P=>P.colonized_by&&P.colonized_by!==this.currentUserId)&&(u+=x*x*.6)}}),o[r][l]=d-u}return o}extractTerritorialContours(e,t,s){const i=[],n=e.length,o=e[0].length,a=new Array(n).fill(null).map(()=>new Array(o).fill(!1)),r=.2;for(let l=0;l<n-1;l++)for(let c=0;c<o-1;c++){if(a[l][c])continue;if(e[l][c]>r){const d=this.traceContour(e,c,l,r,t,s,a);d.length>4&&i.push(d)}}return i}traceContour(e,t,s,i,n,o,a){const r=[],l=e.length,c=e[0].length,h=[{x:t,y:s}],d=new Set;for(;h.length>0;){const{x:u,y:p}=h.shift();u<0||u>=c||p<0||p>=l||a[p][u]||e[p][u]<=i||(a[p][u]=!0,d.add(`${u},${p}`),h.push({x:u+1,y:p},{x:u-1,y:p},{x:u,y:p+1},{x:u,y:p-1}))}return d.forEach(u=>{const[p,f]=u.split(",").map(Number);if([{x:p+1,y:f},{x:p-1,y:f},{x:p,y:f+1},{x:p,y:f-1}].some(y=>y.x<0||y.x>=c||y.y<0||y.y>=l?!0:e[y.y][y.x]<=i)){const y=n.minX+p*o,x=n.minY+f*o,S=this.worldToScreen(y,x);r.push({x:S.x,y:S.y,worldX:y,worldY:x})}}),this.orderContourPoints(r)}orderContourPoints(e){if(e.length<3)return e;const t=e.reduce((i,n)=>i+n.x,0)/e.length,s=e.reduce((i,n)=>i+n.y,0)/e.length;return e.sort((i,n)=>{const o=Math.atan2(i.y-s,i.x-t),a=Math.atan2(n.y-s,n.x-t);return o-a})}drawTerritorialContours(e,t="34, 197, 94",s=!0){e.length!==0&&(this.ctx.save(),this.ctx.globalCompositeOperation="screen",e.forEach(i=>{if(i.length<3)return;this.ctx.beginPath(),this.ctx.moveTo(i[0].x,i[0].y);for(let l=1;l<i.length;l++){const c=i[l],h=i[(l+1)%i.length],d=c.x+(h.x-c.x)*.5,u=c.y+(h.y-c.y)*.5;this.ctx.quadraticCurveTo(c.x,c.y,d,u)}this.ctx.closePath();const n=this.getContourBounds(i),o=this.ctx.createRadialGradient(n.centerX,n.centerY,n.radius*.7,n.centerX,n.centerY,n.radius),r=Math.max(.1,(s?.25:.15)*this.zoom);o.addColorStop(0,`rgba(${t}, ${r*.05})`),o.addColorStop(.7,`rgba(${t}, ${r*.3})`),o.addColorStop(.9,`rgba(${t}, ${r*.6})`),o.addColorStop(1,`rgba(${t}, 0)`),this.ctx.fillStyle=o,this.ctx.fill(),this.ctx.strokeStyle=`rgba(${t}, ${r*.8})`,this.ctx.lineWidth=s?2*this.zoom:1.5*this.zoom,this.ctx.stroke()}),this.ctx.restore())}getContourBounds(e){let t=e[0].x,s=e[0].x,i=e[0].y,n=e[0].y;e.forEach(l=>{t=Math.min(t,l.x),s=Math.max(s,l.x),i=Math.min(i,l.y),n=Math.max(n,l.y)});const o=(t+s)/2,a=(i+n)/2,r=Math.max(s-t,n-i)/2;return{centerX:o,centerY:a,radius:r}}drawFleets(e){!window.gameState||!this.fleets||this.fleets.forEach(s=>{let i,n,o=!1,a=0;const r=window.gameState.fleetOrders&&window.gameState.fleetOrders.find(h=>h.fleet_id===s.id&&(h.status==="pending"||h.status==="processing"));if(r){const h=this.systems.find(u=>u.id===s.current_system),d=this.systems.find(u=>u.id===r.destination_system_id);if(!h||!d){console.warn(`MapRenderer: Could not find from/to system for moving fleet ${s.id}. Order: ${r.id}`);const u=this.systems.find(p=>p.id===s.current_system);if(u)i=u.x+15,n=u.y+15;else return}else{o=!0;const u=window.gameState.currentTick||0,p=r.execute_at_tick||0;let f=r.travel_time_ticks||2;f<=0&&(f=2);const v=p-f,g=u-v,y=Math.max(0,Math.min(1,g/f));i=h.x+(d.x-h.x)*y,n=h.y+(d.y-h.y)*y,a=Math.atan2(d.y-h.y,d.x-h.x),r.route_path&&r.route_path.length>2&&this.drawActiveMultiHopRoute(r,y)}}else if(s.current_system&&s.current_system!==""){const h=this.systems.find(d=>d.id===s.current_system);if(!h){console.warn(`MapRenderer: Could not find current system for stationary fleet ${s.id}`);return}i=h.x+15,n=h.y+15}else{console.warn(`MapRenderer: Fleet ${s.id} has no valid position.`);return}const l=this.worldToScreen(i,n),c=this.selectedFleet&&this.selectedFleet.id===s.id;if(o){this.ctx.fillStyle=c?"#fbbf24":"#8b5cf6",this.ctx.strokeStyle=c?"#f59e0b":"#ffffff",this.ctx.lineWidth=c?3:2;const h=(c?14:12)*this.zoom;this.ctx.save(),this.ctx.translate(l.x,l.y),this.ctx.rotate(a),this.ctx.beginPath(),this.ctx.moveTo(h,0),this.ctx.lineTo(-h/2,h/2),this.ctx.lineTo(-h/2,-h/2),this.ctx.closePath(),this.ctx.fill(),this.ctx.stroke(),this.ctx.restore();const d=(15+Math.sin(Date.now()/200)*5)*this.zoom,u=this.ctx.createRadialGradient(l.x,l.y,0,l.x,l.y,d);u.addColorStop(0,"rgba(139, 92, 246, 0.3)"),u.addColorStop(1,"rgba(139, 92, 246, 0)"),this.ctx.fillStyle=u,this.ctx.beginPath(),this.ctx.arc(l.x,l.y,d,0,Math.PI*2),this.ctx.fill()}else{this.ctx.fillStyle=c?"#fbbf24":this.colors.fleet,this.ctx.strokeStyle=c?"#f59e0b":"#ffffff",this.ctx.lineWidth=c?2:1;const h=(c?10:8)*this.zoom;this.ctx.beginPath(),this.ctx.moveTo(l.x,l.y-h),this.ctx.lineTo(l.x-h,l.y+h),this.ctx.lineTo(l.x+h,l.y+h),this.ctx.closePath(),this.ctx.fill(),this.ctx.stroke(),c&&(this.ctx.strokeStyle="#fbbf24",this.ctx.lineWidth=2,this.ctx.globalAlpha=.7,this.ctx.beginPath(),this.ctx.arc(l.x,l.y,h*2,0,Math.PI*2),this.ctx.stroke(),this.ctx.globalAlpha=1)}this.zoom>.5&&(this.ctx.fillStyle="#ffffff",this.ctx.font=`${10*this.zoom}px Arial`,this.ctx.textAlign="center",this.ctx.fillText(s.name||"Fleet",l.x,l.y-16*this.zoom))})}drawUI(){this.ctx.fillStyle="#ffffff",this.ctx.font="12px monospace",this.ctx.textAlign="left",this.ctx.fillText(`Zoom: ${(this.zoom*100).toFixed(0)}%`,10,25);const e=this.screenToWorld(this.canvas.width/2,this.canvas.height/2);this.ctx.fillText(`Center: ${e.x.toFixed(0)}, ${e.y.toFixed(0)}`,10,45)}setSystems(e){console.log("MapRenderer: Setting systems",e.length,"systems"),this.systems=e,this.updateSystemPlanetCounts(),!this.initialViewSet&&e.length>0&&(this.fitToSystems(),this.initialViewSet=!0),this.isDirty=!0}setHyperlanes(e){console.log("MapRenderer: Setting hyperlanes",e.length,"hyperlanes"),this.hyperlanes=e,this.isDirty=!0}areSystemsConnected(e,t){return!e||!t||e.id===t.id||!this.hyperlanes||this.hyperlanes.length===0?!1:this.hyperlanes.some(s=>s.from_system===e.id&&s.to_system===t.id||s.from_system===t.id&&s.to_system===e.id)}setTrades(e){this.trades=e||[]}updateSystemPlanetCounts(){if(this.systemPlanetCounts.clear(),window.gameState&&window.gameState.planets)for(const e of window.gameState.planets){const t=e.system_id,s=this.systemPlanetCounts.get(t)||0;this.systemPlanetCounts.set(t,s+1)}}setLanes(e){this.lanes=e,this.updateSystemPlanetCounts()}setFleets(e){this.fleets=e,this.isDirty=!0}getFleetAt(e,t){if(!window.gameState||!this.fleets)return null;for(const n of this.fleets){let o,a;const r=window.gameState.fleetOrders&&window.gameState.fleetOrders.find(c=>c.fleet_id===n.id&&(c.status==="pending"||c.status==="processing"));if(r){const c=this.systems.find(d=>d.id===n.current_system),h=this.systems.find(d=>d.id===r.destination_system_id);if(!c||!h){const d=this.systems.find(u=>u.id===n.current_system);if(d)o=d.x+15,a=d.y+15;else continue}else{const d=window.gameState.currentTick||0,u=r.execute_at_tick||0;let p=r.travel_time_ticks||2;p<=0&&(p=2);const f=u-p,v=d-f,g=Math.max(0,Math.min(1,v/p));o=c.x+(h.x-c.x)*g,a=c.y+(h.y-c.y)*g}}else if(n.current_system&&n.current_system!==""){const c=this.systems.find(h=>h.id===n.current_system);if(!c)continue;o=c.x+15,a=c.y+15}else continue;if(Math.sqrt(Math.pow(e-o,2)+Math.pow(t-a,2))<=20/this.zoom)return n}return null}selectFleet(e,t=null,s=null){this.selectedFleet=e,this.isDirty=!0,e&&this.canvas.dispatchEvent(new CustomEvent("fleetSelected",{detail:{fleet:e,screenX:t,screenY:s},bubbles:!0}))}getFleetCurrentSystem(e){return!e||!e.current_system?null:this.systems.find(t=>t.id===e.current_system)}getSelectedFleet(){return this.selectedFleet}setCurrentUserId(e){this.currentUserId!==e&&(this.currentUserId=e,this.territorialCacheKey=null)}setSelectedSystem(e){this.selectedSystem=e,this.updateConnectedSystems()}centerOnSystem(e){const t=this.systems.find(s=>s.id===e);t&&(this.targetViewX=-t.x,this.targetViewY=-t.y)}fitToSystems(){if(this.systems.length===0)return;const e=Math.min(...this.systems.map(h=>h.x)),t=Math.max(...this.systems.map(h=>h.x)),s=Math.min(...this.systems.map(h=>h.y)),i=Math.max(...this.systems.map(h=>h.y)),n=(e+t)/2,o=(s+i)/2,a=t-e+500,r=i-s+500,l=this.canvas.width/a,c=this.canvas.height/r;this.zoom=Math.min(l,c,this.maxZoom),this.zoom>.25&&(this.zoom=.25),this.viewX=-n,this.viewY=-o,this.targetViewX=this.viewX,this.targetViewY=this.viewY}drawActiveMultiHopRoute(e,t=0){if(!e.route_path||!this.systems)return;const s=this.ctx,i=e.route_path,n=e.current_hop||0,o=i.map(a=>this.systems.find(r=>r.id===a)).filter(a=>a!==void 0);if(!(o.length<2)){for(let a=0;a<o.length-1;a++){const r=o[a],l=o[a+1];let c,h,d,u;a<n?(c="#22c55e",h=.4,d=2,u=[]):a===n?(c="#fbbf24",h=.8,d=3,u=[10/this.zoom,5/this.zoom]):(c="#3b82f6",h=.3,d=2,u=[5/this.zoom,5/this.zoom]),s.save(),s.globalAlpha=h,s.strokeStyle=c,s.lineWidth=d/this.zoom,s.setLineDash(u),s.beginPath(),s.moveTo(r.x,r.y),s.lineTo(l.x,l.y),s.stroke(),s.restore()}if(o.forEach((a,r)=>{let l,c,h;r===0?(l="#10b981",c=8,h="START"):r===o.length-1?(l="#ef4444",c=10,h="DEST"):r===n+1?(l="#fbbf24",c=6,h="NEXT"):r<=n?(l="#22c55e",c=4,h=null):(l="#3b82f6",c=4,h=null),s.save(),s.fillStyle=l,s.strokeStyle="#ffffff",s.lineWidth=1/this.zoom,s.beginPath(),s.arc(a.x,a.y,c/this.zoom,0,Math.PI*2),s.fill(),s.stroke(),h&&this.zoom>.4&&(s.font=`${8/this.zoom}px monospace`,s.fillStyle="#ffffff",s.textAlign="center",s.fillText(h,a.x,a.y-12/this.zoom)),s.restore()}),n<o.length-1&&t>0){const a=o[n],r=o[n+1],l=a.x+(r.x-a.x)*t,c=a.y+(r.y-a.y)*t;s.save(),s.fillStyle="#fbbf24",s.strokeStyle="#ffffff",s.lineWidth=2/this.zoom,s.beginPath(),s.arc(l,c,6/this.zoom,0,Math.PI*2),s.fill(),s.stroke(),s.globalAlpha=.3,s.beginPath(),s.arc(l,c,12/this.zoom,0,Math.PI*2),s.fill(),s.restore()}}}destroy(){this.animationFrame&&cancelAnimationFrame(this.animationFrame)}}const Ye="modulepreload",Ge=function(m){return"/"+m},ve={},be=function(e,t,s){let i=Promise.resolve();if(t&&t.length>0){document.getElementsByTagName("link");const o=document.querySelector("meta[property=csp-nonce]"),a=(o==null?void 0:o.nonce)||(o==null?void 0:o.getAttribute("nonce"));i=Promise.allSettled(t.map(r=>{if(r=Ge(r),r in ve)return;ve[r]=!0;const l=r.endsWith(".css"),c=l?'[rel="stylesheet"]':"";if(document.querySelector(`link[href="${r}"]${c}`))return;const h=document.createElement("link");if(h.rel=l?"stylesheet":Ye,l||(h.as="script"),h.crossOrigin="",h.href=r,a&&h.setAttribute("nonce",a),document.head.appendChild(h),l)return new Promise((d,u)=>{h.addEventListener("load",d),h.addEventListener("error",()=>u(new Error(`Unable to preload CSS for ${r}`)))})}))}function n(o){const a=new Event("vite:preloadError",{cancelable:!0});if(a.payload=o,window.dispatchEvent(a),!a.defaultPrevented)throw o}return i.then(o=>{for(const a of o||[])a.status==="rejected"&&n(a.reason);return e().catch(n)})};class Xe{constructor(e,t){this.uiController=e,this.gameState=t,this.currentUser=this.uiController.currentUser}render(){if(!this.gameState||!this.currentUser)return'<div class="text-space-400">Game data not loaded or user not available.</div>';const e=this.getMovingFleets(),t=this.getStationaryFleets();return`
      <div class="fleet-list-container">
        ${e.length>0?this.renderMovingFleets(e):""}
        ${t.length>0?this.renderStationaryFleets(t):""}
        ${e.length===0&&t.length===0?'<div class="text-space-400 text-center py-8">No fleets found.</div>':""}
      </div>
    `}getMovingFleets(){const e=this.gameState.fleetOrders||[],t=this.gameState.fleets||[],s=this.currentUser.id;return e.filter(i=>i.user_id===s&&(i.status==="pending"||i.status==="processing")).map(i=>{const n=t.find(o=>o.id===i.fleet_id);return n?{fleet:n,order:i}:null}).filter(Boolean).sort((i,n)=>i.order.execute_at_tick-n.order.execute_at_tick)}getStationaryFleets(){const e=new Set(this.getMovingFleets().map(t=>t.fleet.id));return this.gameState.getPlayerFleets().filter(t=>!e.has(t.id))}renderMovingFleets(e){const t=e.map(({fleet:s,order:i})=>this.renderMovingFleet(s,i)).join("");return`
      <div class="mb-6">
        <h3 class="text-lg font-semibold mb-3 text-plasma-200 flex items-center gap-2">
          <span class="material-icons text-sm">flight_takeoff</span>
          Moving Fleets (${e.length})
        </h3>
        <div class="space-y-3">
          ${t}
        </div>
      </div>
    `}renderStationaryFleets(e){const t=e.map(s=>this.renderStationaryFleet(s)).join("");return`
      <div>
        <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
          <span class="material-icons text-sm">anchor</span>
          Docked Fleets (${e.length})
        </h3>
        <div class="space-y-3">
          ${t}
        </div>
      </div>
    `}renderMovingFleet(e,t){const s=e.name||`Fleet ${e.id.slice(-4)}`,i=this.gameState.systems||[],n=this.gameState.currentTick||0,a=60/(this.gameState.ticksPerMinute||6),r=i.find(S=>S.id===e.current_system),l=r?r.name:"Deep Space",c=i.find(S=>S.id===t.destination_system_id),h=c?c.name:"Unknown System",d=Math.max(0,t.execute_at_tick-n),u=(d*a).toFixed(0);let p=`${d} ticks (~${u}s)`;d===0&&t.status==="processing"?p="Finalizing Jump":d===0&&t.status==="pending"&&(p="Initiating Jump");const f=t.travel_time_ticks||2,v=f-d,g=Math.round(v/f*100),y=t.status.charAt(0).toUpperCase()+t.status.slice(1);let x="";if(t.route_path&&t.route_path.length>2){const S=t.current_hop||0,C=t.route_path.length-1,k=C-S,P=i.find(I=>I.id===t.final_destination_id);x=`
        <div class="border-t border-space-500 mt-2 pt-2 text-xs">
          <div class="grid grid-cols-2 gap-2">
            <div><span class="text-space-400">Route:</span> <span class="text-purple-400">Multi-hop</span></div>
            <div><span class="text-space-400">Final:</span> ${P?P.name||`System ${P.id.slice(-4)}`:"Unknown"}</div>
            <div><span class="text-space-400">Hop:</span> <span class="text-cyan-400">${S+1}/${C}</span></div>
            <div><span class="text-space-400">Remaining:</span> <span class="text-yellow-400">${k} hops</span></div>
          </div>
        </div>
      `}return`
      <div class="fleet-item bg-space-700 p-4 rounded-lg border border-plasma-600/30 cursor-pointer hover:bg-space-600 transition-all duration-200 shadow-md hover:shadow-lg"
           onclick="window.fleetComponents.showFleetDetails('${e.id}')">
        <div class="flex items-start justify-between mb-2">
          <div class="flex items-center gap-2">
            <span class="material-icons text-plasma-400">rocket_launch</span>
            <div>
              <div class="font-semibold text-plasma-200">${s}</div>
              <div class="text-xs text-space-300">Fleet ID: ${e.id.slice(-8)}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Status</div>
            <div class="text-sm font-medium text-cyan-400">${y}</div>
          </div>
        </div>
        
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div class="text-space-400">From:</div>
            <div class="text-white truncate">${l}</div>
          </div>
          <div>
            <div class="text-space-400">To:</div>
            <div class="text-white truncate">${h}</div>
          </div>
          <div>
            <div class="text-space-400">ETA:</div>
            <div class="text-yellow-400">${p}</div>
          </div>
          <div>
            <div class="text-space-400">Progress:</div>
            <div class="text-green-400">${g}%</div>
          </div>
        </div>

        <div class="w-full bg-space-600 rounded-full h-2 mt-3">
          <div class="bg-gradient-to-r from-plasma-500 to-cyan-500 h-2 rounded-full transition-all duration-300" 
               style="width: ${g}%"></div>
        </div>

        ${x}
      </div>
    `}renderStationaryFleet(e){var c;const t=e.name||`Fleet ${e.id.slice(-4)}`,i=(this.gameState.systems||[]).find(h=>h.id===e.current_system),n=i?i.name:"Deep Space",o=((c=this.gameState)==null?void 0:c.getFleetCargo(e.id))||{cargo:{},used_capacity:0,total_capacity:0},a=e.ships?e.ships.reduce((h,d)=>h+d.count,0):0,r=e.ships?e.ships.length:0;let l="Empty";return o.cargo&&Object.keys(o.cargo).length>0&&(l=`${Object.values(o.cargo).reduce((d,u)=>d+u,0)} units`),`
      <div class="fleet-item bg-space-700 p-4 rounded-lg border border-nebula-600/30 cursor-pointer hover:bg-space-600 transition-all duration-200 shadow-md hover:shadow-lg"
           onclick="window.fleetComponents.showFleetDetails('${e.id}')">
        <div class="flex items-start justify-between mb-3">
          <div class="flex items-center gap-2">
            <span class="material-icons text-nebula-400">anchor</span>
            <div>
              <div class="font-semibold text-nebula-200">${t}</div>
              <div class="text-xs text-space-300">Fleet ID: ${e.id.slice(-8)}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Status</div>
            <div class="text-sm font-medium text-green-400">Docked</div>
          </div>
        </div>
        
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div class="text-space-400">Location:</div>
            <div class="text-white truncate">${n}</div>
          </div>
          <div>
            <div class="text-space-400">Ships:</div>
            <div class="text-white">${a} (${r} types)</div>
          </div>
          <div>
            <div class="text-space-400">Cargo:</div>
            <div class="text-white">${l}</div>
          </div>
          <div>
            <div class="text-space-400">Capacity:</div>
            <div class="text-white">${o.used_capacity}/${o.total_capacity}</div>
          </div>
        </div>

        ${o.total_capacity>0?`
          <div class="mt-3">
            <div class="flex justify-between text-xs mb-1">
              <span class="text-space-400">Cargo Usage</span>
              <span class="text-white">${Math.round(o.used_capacity/o.total_capacity*100)}%</span>
            </div>
            <div class="w-full bg-space-600 rounded-full h-2">
              <div class="bg-gradient-to-r from-nebula-500 to-blue-500 h-2 rounded-full transition-all duration-300" 
                   style="width: ${o.used_capacity/o.total_capacity*100}%"></div>
            </div>
          </div>
        `:""}
      </div>
    `}}class Ke{constructor(e,t){this.uiController=e,this.gameState=t,this.currentUser=this.uiController.currentUser,this.fleet=null}setFleet(e){var t;return this.fleet=(t=this.gameState.fleets)==null?void 0:t.find(s=>s.id===e),this.fleet}render(e){var l;if(!this.setFleet(e))return'<div class="text-red-400">Fleet not found.</div>';const t=this.fleet.name||`Fleet ${this.fleet.id.slice(-4)}`,i=(this.gameState.systems||[]).find(c=>c.id===this.fleet.current_system),n=i?i.name:"Deep Space",o=((l=this.gameState)==null?void 0:l.getFleetCargo(this.fleet.id))||{cargo:{},used_capacity:0,total_capacity:0},a=this.isFleetMoving(),r=a?this.getMovementInfo():null;return`
      <div class="fleet-details-container">
        ${this.renderFleetHeader(t,n,a)}
        ${a?this.renderMovementStatus(r):""}
        ${this.renderFleetStats(o)}
        ${this.renderShipsList()}
        ${this.renderFleetActions()}
      </div>
    `}renderFleetHeader(e,t,s){const i=s?"rocket_launch":"anchor",n=s?"text-plasma-400":"text-nebula-400",o=s?"In Transit":"Docked";return`
      <div class="fleet-header mb-4 p-4 bg-gradient-to-r from-space-800 to-space-700 rounded-lg border border-space-600">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons text-2xl ${n}">${i}</span>
            <div>
              <h2 class="text-xl font-bold text-white">${e}</h2>
              <div class="text-sm text-space-300">Fleet ID: ${this.fleet.id}</div>
              <div class="text-sm text-space-300">Location: ${t}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Status</div>
            <div class="text-sm font-semibold ${n}">${o}</div>
          </div>
        </div>
      </div>
    `}renderMovementStatus(e){if(!e)return"";const{order:t,originName:s,destName:i,etaDisplay:n,progressPercent:o,statusDisplay:a}=e;return`
      <div class="movement-status mb-4 p-4 bg-plasma-900/20 border border-plasma-600/30 rounded-lg">
        <h3 class="text-lg font-semibold mb-3 text-plasma-200 flex items-center gap-2">
          <span class="material-icons text-sm">flight</span>
          Movement Status
        </h3>
        
        <div class="grid grid-cols-2 gap-4 text-sm mb-3">
          <div>
            <div class="text-space-400">From:</div>
            <div class="text-white font-medium">${s}</div>
          </div>
          <div>
            <div class="text-space-400">To:</div>
            <div class="text-white font-medium">${i}</div>
          </div>
          <div>
            <div class="text-space-400">ETA:</div>
            <div class="text-yellow-400 font-medium">${n}</div>
          </div>
          <div>
            <div class="text-space-400">Status:</div>
            <div class="text-cyan-400 font-medium">${a}</div>
          </div>
        </div>

        <div class="mb-2">
          <div class="flex justify-between text-xs mb-1">
            <span class="text-space-400">Progress</span>
            <span class="text-white">${o}%</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-3">
            <div class="bg-gradient-to-r from-plasma-500 to-cyan-500 h-3 rounded-full transition-all duration-300" 
                 style="width: ${o}%"></div>
          </div>
        </div>

        ${this.renderRouteInfo(t)}
      </div>
    `}renderRouteInfo(e){if(!e.route_path||e.route_path.length<=2)return"";const t=e.current_hop||0,s=e.route_path.length-1,i=s-t,o=(this.gameState.systems||[]).find(r=>r.id===e.final_destination_id);return`
      <div class="route-info border-t border-plasma-600/30 mt-3 pt-3">
        <div class="text-sm font-medium text-purple-300 mb-2">Multi-Hop Route</div>
        <div class="grid grid-cols-2 gap-3 text-xs">
          <div><span class="text-space-400">Final Destination:</span> <span class="text-white">${o?o.name||`System ${o.id.slice(-4)}`:"Unknown"}</span></div>
          <div><span class="text-space-400">Current Hop:</span> <span class="text-cyan-400">${t+1}/${s}</span></div>
          <div><span class="text-space-400">Remaining Hops:</span> <span class="text-yellow-400">${i}</span></div>
        </div>
      </div>
    `}renderFleetStats(e){const t=this.fleet.ships?this.fleet.ships.reduce((n,o)=>n+o.count,0):0,s=this.fleet.ships?this.fleet.ships.length:0,i=e.total_capacity>0?Math.round(e.used_capacity/e.total_capacity*100):0;return`
      <div class="fleet-stats mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
          <span class="material-icons text-sm">assessment</span>
          Fleet Statistics
        </h3>
        
        <div class="grid grid-cols-3 gap-4 text-sm">
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-blue-400">${t}</div>
            <div class="text-space-400">Total Ships</div>
          </div>
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-green-400">${s}</div>
            <div class="text-space-400">Ship Types</div>
          </div>
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-purple-400">${i}%</div>
            <div class="text-space-400">Cargo Full</div>
          </div>
        </div>

        ${e.total_capacity>0?`
          <div class="mt-4">
            <div class="flex justify-between text-sm mb-2">
              <span class="text-space-400">Cargo Capacity</span>
              <span class="text-white">${e.used_capacity} / ${e.total_capacity}</span>
            </div>
            <div class="w-full bg-space-600 rounded-full h-3">
              <div class="bg-gradient-to-r from-nebula-500 to-purple-500 h-3 rounded-full transition-all duration-300" 
                   style="width: ${i}%"></div>
            </div>
          </div>
        `:""}
      </div>
    `}renderShipsList(){if(!this.fleet.ships||this.fleet.ships.length===0)return`
        <div class="ships-list mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
            <span class="material-icons text-sm">rocket</span>
            Ships (0)
          </h3>
          <div class="text-space-400 text-center py-4">No ships in this fleet</div>
        </div>
      `;const e=this.fleet.ships.map(t=>this.renderShipItem(t)).join("");return`
      <div class="ships-list mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
          <span class="material-icons text-sm">rocket</span>
          Ships (${this.fleet.ships.length})
        </h3>
        <div class="space-y-2">
          ${e}
        </div>
      </div>
    `}renderShipItem(e){const t=e.ship_type_name||"Unknown",s=e.count||1,i=e.health||100,n=i>75?"text-green-400":i>50?"text-yellow-400":"text-red-400",o=this.getShipTypeData(e),a=(o==null?void 0:o.cargo_capacity)||0,r=a>0;return`
      <div class="ship-item p-3 bg-space-800 rounded border border-space-500 cursor-pointer hover:bg-space-750 transition-colors"
           onclick="window.fleetComponents.showShipDetails('${e.id}')">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons text-cyan-400">${r?"local_shipping":"rocket_launch"}</span>
            <div>
              <div class="font-medium text-white">${s}x ${t}</div>
              <div class="text-xs text-space-300">Ship ID: ${e.id.slice(-8)}</div>
              ${r?`<div class="text-xs text-orange-400">Cargo: ${a} per ship</div>`:""}
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Health</div>
            <div class="text-sm font-medium ${n}">${i}%</div>
            ${r?`
              <button class="btn btn-xs btn-info mt-1" 
                      onclick="event.stopPropagation(); window.fleetComponents.showShipCargo('${e.id}')">
                <span class="material-icons text-xs">inventory_2</span>
              </button>
            `:""}
          </div>
        </div>
        
        <div class="mt-2">
          <div class="w-full bg-space-600 rounded-full h-1.5">
            <div class="${i>75?"bg-green-500":i>50?"bg-yellow-500":"bg-red-500"} h-1.5 rounded-full transition-all duration-300" 
                 style="width: ${i}%"></div>
          </div>
        </div>
      </div>
    `}getShipTypeData(e){return(this.gameState.shipTypes||[]).find(s=>s.id===e.ship_type||s.name===e.ship_type_name)}renderFleetActions(){const e=this.isFleetMoving();return`
      <div class="fleet-actions p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-nebula-200 flex items-center gap-2">
          <span class="material-icons text-sm">settings</span>
          Fleet Actions
        </h3>
        
        <div class="grid grid-cols-2 gap-3">
          <button class="btn btn-primary flex items-center justify-center gap-2" 
                  onclick="window.fleetComponents.sendFleet('${this.fleet.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">send</span>
            Send Fleet
          </button>
          
          <button class="btn btn-secondary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.manageFleet('${this.fleet.id}')">
            <span class="material-icons text-sm">edit</span>
            Manage Fleet
          </button>
          
          <button class="btn btn-info flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.viewCargo('${this.fleet.id}')">
            <span class="material-icons text-sm">inventory</span>
            Fleet Cargo
          </button>
          
          <button class="btn btn-secondary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.back()">
            <span class="material-icons text-sm">arrow_back</span>
            Back to List
          </button>
        </div>
      </div>
    `}isFleetMoving(){return(this.gameState.fleetOrders||[]).some(t=>t.fleet_id===this.fleet.id&&(t.status==="pending"||t.status==="processing"))}getMovementInfo(){const t=(this.gameState.fleetOrders||[]).find(y=>y.fleet_id===this.fleet.id&&(y.status==="pending"||y.status==="processing"));if(!t)return null;const s=this.gameState.systems||[],i=this.gameState.currentTick||0,o=60/(this.gameState.ticksPerMinute||6),a=s.find(y=>y.id===this.fleet.current_system),r=a?a.name:"Deep Space",l=s.find(y=>y.id===t.destination_system_id),c=l?l.name:"Unknown System",h=Math.max(0,t.execute_at_tick-i),d=(h*o).toFixed(0);let u=`${h} ticks (~${d}s)`;h===0&&t.status==="processing"?u="Finalizing Jump":h===0&&t.status==="pending"&&(u="Initiating Jump");const p=t.travel_time_ticks||2,f=p-h,v=Math.round(f/p*100),g=t.status.charAt(0).toUpperCase()+t.status.slice(1);return{order:t,originName:r,destName:c,etaDisplay:u,progressPercent:v,statusDisplay:g}}}class Je{constructor(e,t){this.uiController=e,this.gameState=t,this.currentUser=this.uiController.currentUser,this.ship=null,this.fleet=null}setShip(e){const t=this.gameState.fleets||[];for(const s of t)if(s.ships){const i=s.ships.find(n=>n.id===e);if(i)return this.ship=i,this.fleet=s,i}return null}render(e){if(!this.setShip(e))return'<div class="text-red-400">Ship not found.</div>';const t=this.ship.ship_type_name||"Unknown Ship",s=this.ship.count||1,i=this.fleet.name||`Fleet ${this.fleet.id.slice(-4)}`;return`
      <div class="ship-details-container">
        ${this.renderShipHeader(t,s,i)}
        ${this.renderShipStats()}
        ${this.renderShipCargo()}
        ${this.renderShipActions()}
      </div>
    `}renderShipHeader(e,t,s){const i=this.ship.health||100,n=i>75?"text-green-400":i>50?"text-yellow-400":"text-red-400",o=i>75?"bg-green-500":i>50?"bg-yellow-500":"bg-red-500";return`
      <div class="ship-header mb-4 p-4 bg-gradient-to-r from-space-800 to-space-700 rounded-lg border border-space-600">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons text-2xl text-cyan-400">rocket_launch</span>
            <div>
              <h2 class="text-xl font-bold text-white">${t}x ${e}</h2>
              <div class="text-sm text-space-300">Ship ID: ${this.ship.id}</div>
              <div class="text-sm text-space-300">Fleet: ${s}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Hull Integrity</div>
            <div class="text-lg font-bold ${n}">${i}%</div>
          </div>
        </div>

        <div class="mt-3">
          <div class="flex justify-between text-xs mb-1">
            <span class="text-space-400">Health Status</span>
            <span class="text-white">${this.getHealthStatus(i)}</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-3">
            <div class="${o} h-3 rounded-full transition-all duration-300"
                 style="width: ${i}%"></div>
          </div>
        </div>
      </div>
    `}renderShipStats(){const e=this.getShipTypeData(),t=(e==null?void 0:e.cargo_capacity)||0,s=(e==null?void 0:e.strength)||0;return`
      <div class="ship-stats mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-cyan-200 flex items-center gap-2">
          <span class="material-icons text-sm">assessment</span>
          Ship Specifications
        </h3>

        <div class="grid grid-cols-2 gap-4">
          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-blue-400">inventory</span>
              <span class="text-space-400">Cargo Capacity</span>
            </div>
            <div class="text-xl font-bold text-blue-400">${t}</div>
            <div class="text-xs text-space-500">units per ship</div>
          </div>

          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-red-400">military_tech</span>
              <span class="text-space-400">Combat Strength</span>
            </div>
            <div class="text-xl font-bold text-red-400">${s}</div>
            <div class="text-xs text-space-500">per ship</div>
          </div>

          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-green-400">groups</span>
              <span class="text-space-400">Ship Count</span>
            </div>
            <div class="text-xl font-bold text-green-400">${this.ship.count||1}</div>
            <div class="text-xs text-space-500">in formation</div>
          </div>

          <div class="stat-item p-3 bg-space-800 rounded">
            <div class="flex items-center gap-2 mb-2">
              <span class="material-icons text-sm text-purple-400">calculate</span>
              <span class="text-space-400">Total Strength</span>
            </div>
            <div class="text-xl font-bold text-purple-400">${s*(this.ship.count||1)}</div>
            <div class="text-xs text-space-500">combined</div>
          </div>
        </div>
      </div>
    `}renderShipCargo(){var r;const e=this.getShipCargo(),s=(((r=this.getShipTypeData())==null?void 0:r.cargo_capacity)||0)*(this.ship.count||1);if(s===0)return`
        <div class="ship-cargo mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">inventory_2</span>
            Cargo Hold
          </h3>
          <div class="text-space-400 text-center py-4">This ship type cannot carry cargo</div>
        </div>
      `;const i=e.reduce((l,c)=>l+c.quantity,0),n=s>0?Math.round(i/s*100):0,o=e.length>0?e.slice(0,3).map(l=>this.renderCargoItem(l)).join(""):'<div class="text-space-400 text-center py-4">Cargo hold is empty</div>',a=e.length>3;return`
      <div class="ship-cargo mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-lg font-semibold text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">inventory_2</span>
            Cargo Hold
          </h3>
          <button class="btn btn-sm btn-info flex items-center gap-1"
                  onclick="window.fleetComponents.showShipCargo('${this.ship.id}')">
            <span class="material-icons text-xs">open_in_new</span>
            Detailed View
          </button>
        </div>

        <div class="cargo-capacity mb-4 p-3 bg-space-800 rounded">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-space-400">Capacity Usage</span>
            <span class="text-white">${i} / ${s} units</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-3">
            <div class="bg-gradient-to-r from-orange-500 to-yellow-500 h-3 rounded-full transition-all duration-300"
                 style="width: ${n}%"></div>
          </div>
          <div class="text-center mt-1">
            <span class="text-xs ${n>90?"text-red-400":n>75?"text-yellow-400":"text-green-400"}">
              ${n}% Full
            </span>
          </div>
        </div>

        <div class="cargo-items space-y-2">
          ${o}
          ${a?`
            <div class="text-center py-2">
              <button class="btn btn-sm btn-secondary" onclick="window.fleetComponents.showShipCargo('${this.ship.id}')">
                View ${e.length-3} more cargo types...
              </button>
            </div>
          `:""}
        </div>
      </div>
    `}renderCargoItem(e){const t=this.uiController.getResourceDefinition(e.resource_name||"unknown");return`
      <div class="cargo-item p-3 bg-space-800 rounded border border-space-600 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <span class="material-icons text-xl ${t.color}">${t.icon}</span>
          <div>
            <div class="font-medium text-white">${e.resource_name||"Unknown"}</div>
            <div class="text-xs text-space-300">Resource ID: ${e.resource_type}</div>
          </div>
        </div>
        <div class="text-right">
          <div class="text-lg font-bold text-white">${e.quantity}</div>
          <div class="text-xs text-space-400">units</div>
        </div>
      </div>
    `}renderShipActions(){var s;const e=this.isFleetMoving();return`
      <div class="ship-actions p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-cyan-200 flex items-center gap-2">
          <span class="material-icons text-sm">settings</span>
          Ship Actions
        </h3>

        <div class="grid grid-cols-2 gap-3">
          ${(((s=this.getShipTypeData())==null?void 0:s.cargo_capacity)||0)>0?`
            <button class="btn btn-primary flex items-center justify-center gap-2"
                    onclick="window.fleetComponents.showShipCargo('${this.ship.id}')"
                    ${e?"disabled":""}>
              <span class="material-icons text-sm">inventory_2</span>
              Manage Cargo
            </button>
          `:`
            <button class="btn btn-secondary flex items-center justify-center gap-2" disabled>
              <span class="material-icons text-sm">block</span>
              No Cargo Bay
            </button>
          `}

          <button class="btn btn-secondary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.repairShip('${this.ship.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">build</span>
            Repair Ship
          </button>

          <button class="btn btn-info flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.upgradeShip('${this.ship.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">upgrade</span>
            Upgrade Ship
          </button>

          <button class="btn btn-warning flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.scuttleShip('${this.ship.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">delete_forever</span>
            Scuttle Ship
          </button>
        </div>

        <div class="mt-3 pt-3 border-t border-space-600">
          <button class="btn btn-secondary flex items-center justify-center gap-2 w-full"
                  onclick="window.fleetComponents.backToFleet('${this.fleet.id}')">
            <span class="material-icons text-sm">arrow_back</span>
            Back to Fleet
          </button>
        </div>
      </div>
    `}getHealthStatus(e){return e>=90?"Excellent":e>=75?"Good":e>=50?"Damaged":e>=25?"Heavily Damaged":"Critical"}getShipTypeData(){return(this.gameState.shipTypes||[]).find(t=>t.id===this.ship.ship_type||t.name===this.ship.ship_type_name)}getShipCargo(){var s;const e=(s=this.gameState)==null?void 0:s.getFleetCargo(this.fleet.id);if(!e||!e.cargo)return[];const t=[];for(const[i,n]of Object.entries(e.cargo))n>0&&t.push({ship_id:this.ship.id,resource_name:i,resource_type:i,quantity:n});return t}isFleetMoving(){return(this.gameState.fleetOrders||[]).some(t=>t.fleet_id===this.fleet.id&&(t.status==="pending"||t.status==="processing"))}}class Ze{constructor(e,t){this.uiController=e,this.gameState=t,this.currentUser=this.uiController.currentUser,this.ship=null,this.fleet=null}setShip(e){const t=this.gameState.fleets||[];for(const s of t)if(s.ships){const i=s.ships.find(n=>n.id===e);if(i)return this.ship=i,this.fleet=s,i}return null}render(e){if(!this.setShip(e))return'<div class="text-red-400">Ship not found.</div>';const t=this.ship.ship_type_name||"Unknown Ship",s=this.ship.count||1,i=this.fleet.name||`Fleet ${this.fleet.id.slice(-4)}`;return`
      <div class="ship-cargo-container">
        ${this.renderShipHeader(t,s,i)}
        ${this.renderCargoOverview()}
        ${this.renderCargoDetails()}
        ${this.renderCargoActions()}
      </div>
    `}renderShipHeader(e,t,s){const n=(this.gameState.systems||[]).find(a=>a.id===this.fleet.current_system),o=n?n.name:"Deep Space";return`
      <div class="ship-header mb-4 p-4 bg-gradient-to-r from-space-800 to-space-700 rounded-lg border border-space-600">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <span class="material-icons text-2xl text-orange-400">inventory_2</span>
            <div>
              <h2 class="text-xl font-bold text-white">${t}x ${e}</h2>
              <div class="text-sm text-space-300">Ship ID: ${this.ship.id.slice(-8)}</div>
              <div class="text-sm text-space-300">Fleet: ${s}</div>
              <div class="text-sm text-space-300">Location: ${o}</div>
            </div>
          </div>
          <div class="text-right">
            <div class="text-xs text-space-400">Cargo Management</div>
            <div class="text-sm font-semibold text-orange-400">Detailed View</div>
          </div>
        </div>
      </div>
    `}renderCargoOverview(){const e=this.getShipTypeData(),s=((e==null?void 0:e.cargo_capacity)||0)*(this.ship.count||1),n=this.getShipCargo().reduce((r,l)=>r+l.quantity,0),o=s>0?Math.round(n/s*100):0,a=s-n;return s===0?`
        <div class="cargo-overview mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">info</span>
            Cargo Overview
          </h3>
          <div class="text-center py-6">
            <span class="material-icons text-4xl text-space-500 mb-2">block</span>
            <div class="text-space-400">This ship type cannot carry cargo</div>
            <div class="text-xs text-space-500 mt-1">Combat vessels have no cargo capacity</div>
          </div>
        </div>
      `:`
      <div class="cargo-overview mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
          <span class="material-icons text-sm">assessment</span>
          Cargo Overview
        </h3>
        
        <div class="grid grid-cols-3 gap-4 mb-4">
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-blue-400">${s}</div>
            <div class="text-space-400 text-sm">Total Capacity</div>
          </div>
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-green-400">${n}</div>
            <div class="text-space-400 text-sm">Used Space</div>
          </div>
          <div class="text-center p-3 bg-space-800 rounded">
            <div class="text-2xl font-bold text-purple-400">${a}</div>
            <div class="text-space-400 text-sm">Available</div>
          </div>
        </div>

        <div class="cargo-bar">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-space-400">Capacity Usage</span>
            <span class="text-white">${o}% Full</span>
          </div>
          <div class="w-full bg-space-600 rounded-full h-4 relative overflow-hidden">
            <div class="bg-gradient-to-r from-orange-500 to-yellow-500 h-4 rounded-full transition-all duration-500" 
                 style="width: ${o}%"></div>
            ${o>95?'<div class="absolute inset-0 bg-red-500/20 animate-pulse"></div>':""}
          </div>
          <div class="text-center mt-2">
            <span class="text-xs ${o>95?"text-red-400 font-bold":o>85?"text-yellow-400":"text-green-400"}">
              ${o>95?"OVERLOADED":o>85?"Nearly Full":"Good Capacity"}
            </span>
          </div>
        </div>
      </div>
    `}renderCargoDetails(){const e=this.getShipCargo(),t=this.getShipTypeData();if(((t==null?void 0:t.cargo_capacity)||0)===0)return"";if(e.length===0)return`
        <div class="cargo-details mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">inventory</span>
            Cargo Hold Contents
          </h3>
          <div class="text-center py-8">
            <span class="material-icons text-6xl text-space-500 mb-3">inventory_2</span>
            <div class="text-space-400 text-lg">Cargo hold is empty</div>
            <div class="text-xs text-space-500 mt-2">Use transfer or loading operations to add cargo</div>
          </div>
        </div>
      `;const i=e.map(n=>this.renderDetailedCargoItem(n)).join("");return`
      <div class="cargo-details mb-4 p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
          <span class="material-icons text-sm">inventory</span>
          Cargo Hold Contents (${e.length} types)
        </h3>
        
        <div class="cargo-items space-y-3">
          ${i}
        </div>
      </div>
    `}renderDetailedCargoItem(e){const t=this.uiController.getResourceDefinition(e.resource_name||"unknown"),s=this.getResourceDensity(e.resource_name),i=this.getResourceValue(e.resource_name),n=i*e.quantity;return`
      <div class="cargo-item p-4 bg-space-800 rounded-lg border border-space-600 hover:border-space-500 transition-colors">
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="resource-icon p-3 bg-space-700 rounded-lg">
              <span class="material-icons text-2xl ${t.color}">${t.icon}</span>
            </div>
            <div class="flex-1">
              <div class="flex items-center gap-2 mb-1">
                <h4 class="text-lg font-semibold text-white">${e.resource_name||"Unknown"}</h4>
                <span class="px-2 py-1 bg-space-600 rounded text-xs text-space-300">${e.resource_type.slice(-4)}</span>
              </div>
              <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span class="text-space-400">Quantity:</span>
                  <span class="text-white font-medium ml-2">${e.quantity} units</span>
                </div>
                <div>
                  <span class="text-space-400">Density:</span>
                  <span class="text-white font-medium ml-2">${s} kg/unit</span>
                </div>
                <div>
                  <span class="text-space-400">Unit Value:</span>
                  <span class="text-green-400 font-medium ml-2">${i} credits</span>
                </div>
                <div>
                  <span class="text-space-400">Total Value:</span>
                  <span class="text-green-400 font-bold ml-2">${n} credits</span>
                </div>
              </div>
            </div>
          </div>
          <div class="flex flex-col gap-2">
            <button class="btn btn-sm btn-info flex items-center gap-1" 
                    onclick="window.fleetComponents.transferCargoType('${this.ship.id}', '${e.resource_type}')"
                    ${this.isFleetMoving()?"disabled":""}>
              <span class="material-icons text-xs">swap_horiz</span>
              Transfer
            </button>
            <button class="btn btn-sm btn-warning flex items-center gap-1"
                    onclick="window.fleetComponents.jettison('${this.ship.id}', '${e.resource_type}')"
                    ${this.isFleetMoving()?"disabled":""}>
              <span class="material-icons text-xs">launch</span>
              Jettison
            </button>
          </div>
        </div>
      </div>
    `}renderCargoActions(){const e=this.isFleetMoving(),t=this.getShipTypeData();return((t==null?void 0:t.cargo_capacity)||0)===0?`
        <div class="cargo-actions p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
            <span class="material-icons text-sm">build</span>
            Ship Actions
          </h3>
          
          <div class="text-center py-4">
            <div class="text-space-400 mb-4">No cargo operations available for this ship type</div>
            <button class="btn btn-secondary flex items-center justify-center gap-2 mx-auto"
                    onclick="window.fleetComponents.backToShip('${this.ship.id}')">
              <span class="material-icons text-sm">arrow_back</span>
              Back to Ship Details
            </button>
          </div>
        </div>
      `:`
      <div class="cargo-actions p-4 bg-space-700 rounded-lg border border-space-600">
        <h3 class="text-lg font-semibold mb-3 text-orange-200 flex items-center gap-2">
          <span class="material-icons text-sm">build</span>
          Cargo Operations
        </h3>
        
        <div class="grid grid-cols-2 gap-3">
          <button class="btn btn-primary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.loadCargo('${this.ship.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">download</span>
            Load Cargo
          </button>
          
          <button class="btn btn-secondary flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.unloadCargo('${this.ship.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">upload</span>
            Unload Cargo
          </button>
          
          <button class="btn btn-info flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.transferAllCargo('${this.ship.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">compare_arrows</span>
            Transfer All
          </button>
          
          <button class="btn btn-warning flex items-center justify-center gap-2"
                  onclick="window.fleetComponents.jettisonAll('${this.ship.id}')"
                  ${e?"disabled":""}>
            <span class="material-icons text-sm">delete_sweep</span>
            Jettison All
          </button>
        </div>

        <div class="mt-4 pt-3 border-t border-space-600">
          <button class="btn btn-secondary flex items-center justify-center gap-2 w-full"
                  onclick="window.fleetComponents.backToShip('${this.ship.id}')">
            <span class="material-icons text-sm">arrow_back</span>
            Back to Ship Details
          </button>
        </div>
      </div>
    `}getShipTypeData(){return(this.gameState.shipTypes||[]).find(t=>t.id===this.ship.ship_type||t.name===this.ship.ship_type_name)}getShipCargo(){var s;const e=(s=this.gameState)==null?void 0:s.getFleetCargo(this.fleet.id);if(!e||!e.cargo)return[];const t=[];for(const[i,n]of Object.entries(e.cargo))n>0&&t.push({ship_id:this.ship.id,resource_name:i,resource_type:i,quantity:n});return t}getResourceDensity(e){return{ore:2.5,metal:7.8,fuel:.8,food:1.2,water:1,machinery:3.2,electronics:1.5}[e==null?void 0:e.toLowerCase()]||1}getResourceValue(e){return{ore:10,metal:25,fuel:15,food:8,water:5,machinery:100,electronics:200}[e==null?void 0:e.toLowerCase()]||1}isFleetMoving(){return(this.gameState.fleetOrders||[]).some(t=>t.fleet_id===this.fleet.id&&(t.status==="pending"||t.status==="processing"))}}class Qe{constructor(e,t){this.uiController=e,this.gameState=t,this.fleetListComponent=new Xe(e,t),this.fleetComponent=new Ke(e,t),this.shipComponent=new Je(e,t),this.shipCargoComponent=new Ze(e,t),this.currentView="list",this.currentFleetId=null,this.currentShipId=null,window.fleetComponents=this}showFleetPanel(){this.currentView="list",this.currentFleetId=null,this.currentShipId=null;const e=this.fleetListComponent.render(),t=this.getFleetCount();this.uiController.showModal(t,e)}showFleetDetails(e){var n;this.currentView="fleet",this.currentFleetId=e,this.currentShipId=null;const t=this.fleetComponent.render(e),s=(n=this.gameState.fleets)==null?void 0:n.find(o=>o.id===e),i=s?s.name||`Fleet ${s.id.slice(-4)}`:"Unknown Fleet";this.uiController.showModal(`Fleet Details: ${i}`,t)}showShipDetails(e){this.currentView="ship",this.currentShipId=e;const t=this.shipComponent.render(e),s=this.findShip(e),i=s?`${s.count||1}x ${s.ship_type_name||"Unknown"}`:"Unknown Ship";this.uiController.showModal(`Ship Details: ${i}`,t)}showShipCargo(e){this.currentView="cargo",this.currentShipId=e;const t=this.shipCargoComponent.render(e),s=this.findShip(e),i=s?`${s.count||1}x ${s.ship_type_name||"Unknown"}`:"Unknown Ship";this.uiController.showModal(`Cargo Management: ${i}`,t)}back(){switch(this.currentView){case"cargo":this.currentShipId?this.showShipDetails(this.currentShipId):this.showFleetPanel();break;case"ship":this.currentFleetId?this.showFleetDetails(this.currentFleetId):this.showFleetPanel();break;case"fleet":this.showFleetPanel();break;default:this.uiController.hideModal();break}}backToFleet(e){this.showFleetDetails(e)}backToShip(e){this.showShipDetails(e)}sendFleet(e){var s;const t=(s=this.gameState.fleets)==null?void 0:s.find(i=>i.id===e);if(t){const n=(this.gameState.systems||[]).find(o=>o.id===t.current_system);n&&(this.uiController.hideModal(),this.uiController.showSendFleetModal(n))}}manageFleet(e){this.uiController.showToast("Fleet management coming soon!","info",3e3)}viewCargo(e){var s;const t=(s=this.gameState.fleets)==null?void 0:s.find(i=>i.id===e);if(t&&t.ships&&t.ships.length>0){const i=t.ships.find(n=>{const o=this.getShipTypeData(n);return(o==null?void 0:o.cargo_capacity)>0});i?this.showShipCargo(i.id):this.uiController.showToast("No cargo ships found in this fleet","info",3e3)}else this.uiController.showToast("Fleet has no ships","error",3e3)}transferCargo(e){this.showShipCargo(e)}repairShip(e){this.uiController.showToast("Ship repair coming soon!","info",3e3)}scuttleShip(e){const t=this.findShip(e);if(t){const s=`${t.count||1}x ${t.ship_type_name||"Unknown"}`;confirm(`Are you sure you want to scuttle ${s}? This action cannot be undone.`)&&this.uiController.showToast("Ship scuttling coming soon!","info",3e3)}}refresh(){switch(this.currentView){case"list":this.showFleetPanel();break;case"fleet":this.currentFleetId&&this.showFleetDetails(this.currentFleetId);break;case"ship":this.currentShipId&&this.showShipDetails(this.currentShipId);break;case"cargo":this.currentShipId&&this.showShipCargo(this.currentShipId);break}}getFleetCount(){var i,n;const e=((n=(i=this.gameState).getPlayerFleets)==null?void 0:n.call(i))||[],t=e.length,s=e.reduce((o,a)=>o+(a.ships?a.ships.reduce((r,l)=>r+(l.count||1),0):0),0);return`Your Fleets (${t} fleets, ${s} ships)`}findShip(e){const t=this.gameState.fleets||[];for(const s of t)if(s.ships){const i=s.ships.find(n=>n.id===e);if(i)return i}return null}loadCargo(e){this.uiController.showToast("Load cargo functionality coming soon!","info",3e3)}unloadCargo(e){this.uiController.showToast("Unload cargo functionality coming soon!","info",3e3)}transferAllCargo(e){this.uiController.showToast("Transfer all cargo functionality coming soon!","info",3e3)}transferCargoType(e,t){this.uiController.showToast("Transfer specific cargo type functionality coming soon!","info",3e3)}jettison(e,t){this.findShip(e)&&confirm("Are you sure you want to jettison this cargo? This action cannot be undone.")&&this.uiController.showToast("Jettison cargo functionality coming soon!","info",3e3)}jettisonAll(e){this.findShip(e)&&confirm("Are you sure you want to jettison ALL cargo? This action cannot be undone.")&&this.uiController.showToast("Jettison all cargo functionality coming soon!","info",3e3)}upgradeShip(e){this.uiController.showToast("Ship upgrade functionality coming soon!","info",3e3)}getShipTypeData(e){return(this.gameState.shipTypes||[]).find(s=>s.id===e.ship_type||s.name===e.ship_type_name)}updateGameState(e){this.gameState=e,this.fleetListComponent.gameState=e,this.fleetComponent.gameState=e,this.shipComponent.gameState=e,this.shipCargoComponent.gameState=e}}class et{constructor(){this.currentUser=null,this.gameState=null,this.tickTimer=null,this.currentSystemId=null,this.planetTypes=new Map,this.pb=null,this.resourceTypes=new Map,this.fleetComponentManager=null,this.displayedResources=this.loadResourcePreferences(),window.uiController=this,this.expandedView=document.getElementById("expanded-view-container"),this.expandedView?(this.expandedView.classList.add("hidden","floating-panel"),this.expandedView.style.left="-2000px",this.expandedView.style.top="-2000px"):console.error("#expanded-view-container not found during UIController construction"),this.initializeResourcesDropdown()}setPocketBase(e){this.pb=e,this.loadPlanetTypes(),this.loadResourceTypes()}async loadPlanetTypes(){try{if(!this.pb)return;const e=await this.pb.collection("planet_types").getFullList();this.planetTypes.clear(),e.forEach(t=>{const s={name:t.name,icon:t.icon||""};this.planetTypes.set(t.name.toLowerCase(),s),this.planetTypes.set(t.id,s)})}catch(e){console.warn("Failed to load planet types:",e)}}async loadResourceTypes(){try{if(!this.pb)return;const e=await this.pb.send("/api/resource_types",{method:"GET"});this.resourceTypes.clear(),e.items.forEach(t=>{const s={name:t.name,icon:t.icon||""};this.resourceTypes.set(t.name.toLowerCase(),s),this.resourceTypes.set(t.id,s)}),this.updateResourcesDropdown()}catch(e){console.warn("Failed to load resource types:",e)}}getPlanetTypeIcon(e){if(!e)return'<img src="/placeholder-planet.svg" class="w-6 h-6" alt="Unknown planet type" />';let t=this.planetTypes.get(e);if(t||(t=this.planetTypes.get(e.toLowerCase())),t&&t.icon){const i={highlands:"border-green-400",abundant:"border-emerald-400",fertile:"border-lime-400",mountain:"border-stone-400",desert:"border-yellow-400",volcanic:"border-red-400",swamp:"border-blue-400",barren:"border-gray-400",radiant:"border-purple-400",barred:"border-red-600"}[t.name.toLowerCase()]||"border-space-300";return`<img src="${t.icon}" class="w-6 h-6 rounded border-2 ${i}" alt="${t.name}" title="${t.name}" />`}return'<img src="/placeholder-planet.svg" class="w-6 h-6" alt="Unknown planet type" />'}getPlanetTypeName(e){if(!e)return"Unknown";let t=this.planetTypes.get(e);return t||(t=this.planetTypes.get(e.toLowerCase())),t?t.name:"Unknown"}getPlanetAnimatedGif(e){if(!e)return null;const t="/planets/default.gif",s={highlands:"/planets/highlands.gif",abundant:"/planets/abundant.gif",fertile:"/planets/fertile.gif",mountain:"/planets/mountain.gif",desert:"/planets/desert.gif",volcanic:"/planets/volcanic.gif",swamp:"/planets/swamp.gif",barren:"/planets/barren.gif",radiant:"/planets/radiant.gif",barred:"/planets/barred.gif",null:"/planets/null.gif"};let i=this.planetTypes.get(e);i||(i=this.planetTypes.get(e.toLowerCase()));const n=i?i.name.toLowerCase():e.toLowerCase();return`<img src="${s[n]||t}" class="w-12 h-12 rounded-full border-2 border-space-400 shadow-lg" alt="${n} planet" title="${n} planet" />`}getPlanetTypeGradient(e){if(!e)return"from-nebula-900/30 to-plasma-900/30";let t=this.planetTypes.get(e);return t||(t=this.planetTypes.get(e.toLowerCase())),t&&{highlands:"from-green-900/30 to-emerald-800/30",abundant:"from-green-800/30 to-lime-700/30",fertile:"from-green-700/30 to-green-600/30",mountain:"from-gray-800/30 to-slate-700/30",desert:"from-yellow-800/30 to-orange-700/30",volcanic:"from-red-900/30 to-orange-800/30",swamp:"from-cyan-900/30 to-teal-800/30",barren:"from-gray-900/30 to-gray-800/30",radiant:"from-yellow-600/30 to-amber-500/30",barred:"from-red-800/30 to-red-900/30"}[t.name.toLowerCase()]||"from-nebula-900/30 to-plasma-900/30"}getSystemGradient(e){if(!e||e.length===0)return"from-nebula-900/30 to-plasma-900/30";const t={};e.forEach(o=>{const a=o.planet_type||o.type;if(a){let r=this.planetTypes.get(a);if(r||(r=this.planetTypes.get(a.toLowerCase())),r){const l=r.name.toLowerCase();t[l]=(t[l]||0)+1}}});let s="unknown",i=0;for(const[o,a]of Object.entries(t))a>i&&(i=a,s=o);return{highlands:"from-green-900/30 to-emerald-800/30",abundant:"from-green-800/30 to-lime-700/30",fertile:"from-green-700/30 to-green-600/30",mountain:"from-gray-800/30 to-slate-700/30",desert:"from-yellow-800/30 to-orange-700/30",volcanic:"from-red-900/30 to-orange-800/30",swamp:"from-cyan-900/30 to-teal-800/30",barren:"from-gray-900/30 to-gray-800/30",radiant:"from-yellow-600/30 to-amber-500/30",barred:"from-red-800/30 to-red-900/30"}[s]||"from-nebula-900/30 to-plasma-900/30"}async getResourceNodes(e){if(!this.pb)return[];try{return await this.pb.collection("resource_nodes").getFullList({filter:`planet_id = "${e}"`,expand:"resource_type"})}catch(t){return console.warn("Failed to load resource nodes:",t),[]}}async loadResourceTypes(){if(!this.pb)return[];try{return await this.pb.collection("resource_types").getFullList()}catch(e){return console.warn("Failed to load resource types:",e),[]}}async getResourceIcons(e){const t=[];if(e.resourceNodes&&e.resourceNodes.length>0){const s=await this.loadResourceTypes(),i={};s.forEach(o=>{i[o.id]=o});const n=new Set;e.resourceNodes.forEach(o=>{let a;if(o.expand&&o.expand.resource_type?a=o.expand.resource_type:a=i[o.resource_type],a){const r=a.name.toLowerCase();if(!n.has(r)){n.add(r);const l=a.icon||"/placeholder-planet.svg",c=a.name;t.push(`<img src="${l}" class="w-5 h-5" title="${c}" alt="${c}" />`)}}})}return t}clearExpandedView(){if(this.expandedView){if(this.expandedView._isPinned)return;this.expandedView._dragCleanup&&(this.expandedView._dragCleanup(),delete this.expandedView._dragCleanup),this.expandedView.classList.remove("focused","pinned-panel","glass-panel"),this.expandedView.classList.add("hidden"),this.expandedView.style.left="-9999px",this.expandedView.style.top="-9999px"}}positionPanel(e,t,s){const i=e.classList.contains("hidden");i&&(e.classList.remove("hidden"),e.style.left="-9999px",e.style.top="-9999px");const n=e.offsetWidth,o=e.offsetHeight,a=window.innerWidth,r=window.innerHeight,l=20,c=120;let h=s-c,d=t+c;const u=[{left:t+c,top:s-c},{left:t-n-c,top:s-c},{left:t+c,top:s+c},{left:t-n-c,top:s+c},{left:t+c,top:s-o/2},{left:t-n-c,top:s-o/2}];let p=u[0];for(const f of u)if(f.left>=l&&f.left+n+l<=a&&f.top>=l&&f.top+o+l<=r){p=f;break}d=p.left,h=p.top,d<l&&(d=l),d+n+l>a&&(d=a-n-l),h<l&&(h=l),h+o+l>r&&(h=r-o-l),e.style.left=`${d}px`,e.style.top=`${h}px`,i&&e.style.left==="-9999px"&&e.classList.add("hidden")}async displaySystemView(e,t,s,i){var h;const n=this.expandedView;if(!n){console.error("#expanded-view-container not found in displaySystemView");return}this.currentSystemId===e.id&&n.classList.contains("hidden"),n.classList.remove("hidden"),n.classList.add("floating-panel","glass-panel"),this.currentSystemId=e.id;let o=0;const a=(h=this.currentUser)==null?void 0:h.id;t&&t.length>0&&t.forEach(d=>{d.colonized_by&&d.colonized_by,o+=d.Pop||0}),t&&t.length>0&&t.map(d=>{const u=d.name||`Planet ${d.id.slice(-4)}`,p=d.planet_type||d.type,f=this.getPlanetTypeIcon(p),v=this.getPlanetTypeName(p),g=d.colonized_by===a;d.Pop,d.MaxPopulation;let y="";d.colonized_by?g?y='<span class="text-xs text-green-400 flex items-center gap-1"><span class="material-icons text-xs">check_circle</span>Your Colony</span>':y=`<span class="text-xs text-red-400 flex items-center gap-1"><span class="material-icons text-xs">block</span>${d.colonized_by_name||"Occupied"}</span>`:y='<span class="text-xs text-gray-400 flex items-center gap-1"><span class="material-icons text-xs">radio_button_unchecked</span>Uncolonized</span>';const x=this.getResourceIcons(d);let S="";return x.length>0&&(S=`
            <div class="mt-2 flex gap-1 items-center">
              ${x.join("")}
            </div>
          `),`
          <li class="mb-2 p-3 bg-space-700 hover:bg-space-600 rounded-md cursor-pointer transition-all duration-200 border border-transparent hover:border-space-500"
              onclick="window.uiController.displayPlanetView(JSON.parse(decodeURIComponent('${encodeURIComponent(JSON.stringify(d))}')))">
            <div class="flex items-start justify-between">
              <div class="flex-1">
                <div class="flex items-center gap-2">
                  <div class="flex items-center justify-center w-8 h-8">${f}</div>
                  <div>
                    <div class="font-semibold">${u}</div>
                    <div class="text-xs text-space-300">${v} • Size ${d.size||"N/A"}</div>
                  </div>
                </div>
                ${S}
              </div>
              <div class="text-right">
                ${y}
              </div>
            </div>
          </li>
        `}).join(""),n.innerHTML=`
      <div class="floating-panel-content">
        <div id="system-header" class="panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r from-nebula-900/30 to-plasma-900/30" draggable="false">
          <div class="flex items-center gap-2">
            <span class="material-icons text-space-400 drag-handle">drag_indicator</span>
            <span id="system-name" class="text-xl font-bold text-nebula-200"></span>
          </div>
          <div class="flex items-center gap-4">
            <div class="text-right">
              <div id="system-seed" class="font-semibold text-nebula-200 text-sm"></div>
              <div id="system-coords" class="font-mono text-xs text-gray-500"></div>
            </div>
            <div class="flex items-center gap-2">
              <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                      class="pin-button text-space-400 hover:text-white transition-colors" 
                      title="Pin panel">
                <span class="material-icons text-sm">push_pin</span>
              </button>
              <button onclick="window.uiController.clearExpandedView()"
                      class="btn-icon hover:bg-space-700 rounded">
                <span class="material-icons text-sm">close</span>
              </button>
            </div>
          </div>
        </div>
        <div class="p-4">

          <div class="flex-1 overflow-hidden flex flex-col">
            <div class="flex justify-end mb-2">
              <div class="text-xs text-space-400">
                <kbd class="px-1 py-0.5 bg-space-700 rounded text-xs">Click</kbd> Fleet to Select • <kbd class="px-1 py-0.5 bg-space-700 rounded text-xs">Shift+Click</kbd> System to Move • <kbd class="px-1 py-0.5 bg-space-700 rounded text-xs">↑↓←→</kbd> Navigate
              </div>
            </div>
            <ul id="system-planets-list" class="flex-1 overflow-y-auto pr-2 custom-scrollbar">
            </ul>
          </div>
        </div>
      </div>
      `;const r=this.getSystemGradient(t),l=n.querySelector("#system-header");l.className=`panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r ${r}`,n.querySelector("#system-name").textContent=e.name||`System ${e.id.slice(-4)}`,n.querySelector("#system-seed").textContent=`Seed: ${e.id.slice(-8)}`,n.querySelector("#system-coords").textContent=`${e.x}, ${e.y}`;const c=n.querySelector("#system-planets-list");await this.updatePlanetList(c,t,a),n.dataset.viewType="system",n.dataset.currentId=e.id,n.classList.remove("hidden"),n._isPinned||(s!==void 0&&i!==void 0?this.positionPanel(n,s,i):(n.style.left==="-2000px"||n.style.left==="-9999px"||!n.style.left)&&(n.style.top="20px",n.style.left="20px",n.style.right="auto")),this.makePanelDraggable(n),this.addPanelFocusEffects(n)}makePanelDraggable(e){const t=e.querySelector(".panel-header");if(!t)return;e._dragCleanup&&e._dragCleanup();let s=!1,i,n,o,a;const r=d=>{if(d.target.closest(".panel-header")&&!d.target.closest("button")){const u=e.getBoundingClientRect();i=u.left,n=u.top,d.type==="touchstart"?(o=d.touches[0].clientX-i,a=d.touches[0].clientY-n):(o=d.clientX-i,a=d.clientY-n),s=!0,e.style.transition="none",e.style.right="auto",t.style.cursor="grabbing",d.preventDefault()}},l=()=>{s&&(s=!1,e.style.transition="",t.style.cursor="move")},c=d=>{if(s){d.preventDefault(),d.type==="touchmove"?(i=d.touches[0].clientX-o,n=d.touches[0].clientY-a):(i=d.clientX-o,n=d.clientY-a);const u=e.getBoundingClientRect(),p=window.innerWidth-u.width,f=window.innerHeight-u.height;i=Math.max(0,Math.min(i,p)),n=Math.max(0,Math.min(n,f)),e.style.left=`${i}px`,e.style.top=`${n}px`}};t.addEventListener("mousedown",r),document.addEventListener("mousemove",c),document.addEventListener("mouseup",l),t.addEventListener("touchstart",r),document.addEventListener("touchmove",c),document.addEventListener("touchend",l);const h=()=>{t.removeEventListener("mousedown",r),document.removeEventListener("mousemove",c),document.removeEventListener("mouseup",l),t.removeEventListener("touchstart",r),document.removeEventListener("touchmove",c),document.removeEventListener("touchend",l)};e._dragCleanup=h}async updatePlanetList(e,t,s){if(!t||t.length===0){e.innerHTML='<div class="text-sm text-space-400">No planets detected in this system.</div>';return}e.innerHTML="";for(const i of t){const n=await this.createEmbeddedPlanetContainer(i,s);e.appendChild(n)}}async createEmbeddedPlanetContainer(e,t){const s=e.id,i=e.name||`Planet ${e.id.slice(-4)}`,n=e.planet_type||e.type,o=this.getPlanetTypeName(n),a=this.getPlanetAnimatedGif(o),r=e.colonized_by===t,l=e.Pop||0,c=e.MaxPopulation||"N/A",h=await this.getResourceNodes(e.id);e.resourceNodes=h;const d=await this.getResourceIcons(e);let u="";e.colonized_by?r?u='<span class="text-xs text-green-400 flex items-center gap-1"><span class="material-icons text-xs">check_circle</span>Your Colony</span>':u=`<span class="text-xs text-red-400 flex items-center gap-1"><span class="material-icons text-xs">block</span>${e.colonized_by_name||"Occupied"}</span>`:u='<span class="text-xs text-gray-400 flex items-center gap-1"><span class="material-icons text-xs">radio_button_unchecked</span>Uncolonized</span>';const p=document.createElement("div");p.className="mb-3 p-3 bg-space-700/30 border border-space-600/50 rounded-lg hover:bg-space-650/40 transition-all duration-200 cursor-pointer",p.dataset.planetId=s,p.innerHTML=`
      <div class="flex items-start gap-4">
        <!-- Planet Icon -->
        <div class="flex-shrink-0">
          <div class="planet-icon-container w-16 h-16 flex items-center justify-center">
            <!-- GIF will be set via DOM manipulation -->
          </div>
        </div>

        <!-- Planet Info -->
        <div class="flex-1 min-w-0">
          <div class="flex items-start justify-between mb-2">
            <div>
              <h3 class="font-semibold text-lg text-nebula-200">${i}</h3>
              <div class="text-sm text-space-300">${o} • Size ${e.size||"N/A"}</div>
            </div>
            <div class="text-right">
              ${u}
              ${l>0?`<div class="text-sm text-green-400 mt-1">${l.toLocaleString()}/${c} pop</div>`:""}
            </div>
          </div>

          <!-- Resources -->
          ${d.length>0?`
            <div class="mb-2">
              <div class="text-xs text-space-400 mb-1">Resources:</div>
              <div class="flex items-center gap-1 flex-wrap">
                ${d.join("")}
              </div>
            </div>
          `:'<div class="text-xs text-space-500 mb-2">No resources detected</div>'}


        </div>
      </div>
    `;const f=p.querySelector(".planet-icon-container");return f&&a&&(f.innerHTML=a),p.onclick=()=>{window.uiController.displayPlanetView(e)},p}async displayPlanetView(e,t,s){var ee,te,se,ie,ne,oe,ae,re,le,ce,de;const i=this.expandedView;if(!i){console.error("#expanded-view-container not found in displayPlanetView");return}const n=await this.getResourceNodes(e.id);e.resourceNodes=n,this.currentSystemId===e.id&&i.dataset.viewType==="planet"&&i.classList.contains("hidden"),i.className="floating-panel",this.currentSystemId=e.id,i.dataset.viewType="planet",i.dataset.currentId=e.id;const o=e.name||`Planet ${e.id.slice(-4)}`,a=e.planet_type||e.type,r=this.getPlanetTypeIcon(a),l=this.getPlanetTypeName(a),c=e.system_name||this.gameState&&((ee=this.gameState.mapData.systems.find(E=>E.id===e.system_id))==null?void 0:ee.name)||e.system_id,h=e.MaxPopulation?Math.round((e.Pop||0)/e.MaxPopulation*100):0,d=h>80?"bg-green-500":h>50?"bg-yellow-500":"bg-orange-500";e.Credits!==void 0&&`${((te=e.Pop)==null?void 0:te.toLocaleString())||0}${((se=e.MaxPopulation)==null?void 0:se.toLocaleString())||"N/A"}${d}${h}${e.Morale||0}${e.Morale||0}${((ie=e.Credits)==null?void 0:ie.toLocaleString())||0}${((ne=e.Food)==null?void 0:ne.toLocaleString())||0}${((oe=e.Ore)==null?void 0:oe.toLocaleString())||0}${((ae=e.Goods)==null?void 0:ae.toLocaleString())||0}${((re=e.Fuel)==null?void 0:re.toLocaleString())||0}`;let u='<div class="text-sm text-space-400">No buildings constructed.</div>',p=[];if(e.Buildings&&Object.keys(e.Buildings).length>0){const E=((ce=(le=this.gameState)==null?void 0:le.populations)==null?void 0:ce.filter(L=>L.planet_id===e.id))||[],N=new Set(E.map(L=>L.employed_at).filter(Boolean)),Y=Object.entries(e.Buildings).map(([L,F])=>{var X,ue,pe;let _=L,R="🏢",z=L;const W=(ue=(X=this.gameState)==null?void 0:X.buildings)==null?void 0:ue.find(j=>j.planet_id===e.id&&(j.building_type===L||j.id===L));if(W&&(z=W.id),this.gameState&&this.gameState.buildingTypes){const j=this.gameState.buildingTypes.find(me=>me.id===L||me.name.toLowerCase()===L.toLowerCase());j&&(_=j.name,_.toLowerCase().includes("farm")?R="🌾":_.toLowerCase().includes("mine")?R="⛏️":_.toLowerCase().includes("factory")?R="🏭":_.toLowerCase().includes("bank")?R="🏦":_.toLowerCase().includes("research")&&(R="🔬"))}const V=N.has(z),G=((pe=E.find(j=>j.employed_at===z))==null?void 0:pe.count)||0;V||p.push(_);const he=V?"":'<span class="text-yellow-400 text-sm ml-1" title="No workers assigned">⚠️</span>',J=V?`<div class="text-xs text-green-400">${G} workers</div>`:'<div class="text-xs text-yellow-400">No workers</div>';return`
          <li class="p-3 ${V?"bg-space-700":"bg-yellow-900/20 border border-yellow-600/30"} rounded-md flex items-center justify-between hover:bg-space-600 transition-colors">
            <div class="flex items-center gap-2">
              <span class="text-xl">${R}</span>
              <div>
                <div class="flex items-center">
                  <span class="font-semibold">${_}</span>
                  ${he}
                </div>
                ${J}
              </div>
            </div>
            <span class="text-sm text-space-300">Level ${F}</span>
          </li>
        `}).join("");let B="";p.length>0&&(B=`
        <div class="p-3 bg-yellow-900/20 border border-yellow-600/50 rounded-md mb-3">
          <div class="flex items-center gap-2 mb-2">
            <span class="text-yellow-400">⚠️</span>
            <span class="font-semibold text-yellow-200">Production Warning</span>
          </div>
          <div class="text-sm text-yellow-100">
            ${p.length} building${p.length>1?"s":""} without workers: 
            <strong>${p.join(", ")}</strong>
          </div>
          <div class="text-xs text-yellow-300 mt-1">
            Buildings without assigned population will not produce resources.
          </div>
        </div>`),u=`${B}<ul class="space-y-2">${Y}</ul>`}const f=e.colonized_by&&e.colonized_by!=="",v=f&&e.colonized_by===((de=this.currentUser)==null?void 0:de.id),g=!f&&this.currentUser&&this.gameState;i.innerHTML=`
      <div class="floating-panel-content">
        <div id="planet-header" class="panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r from-nebula-900/30 to-plasma-900/30" draggable="false">
          <div class="flex items-center gap-2">
            <span class="material-icons text-space-400 drag-handle">drag_indicator</span>
            <span id="planet-icon" class="text-2xl"></span>
            <span id="planet-name" class="text-xl font-bold text-nebula-200"></span>
          </div>
          <div class="flex items-center gap-4">
            <div class="text-right">
              <div id="planet-seed" class="font-semibold text-nebula-200 text-sm"></div>
              <div id="planet-system" class="font-mono text-xs text-gray-500"></div>
            </div>
            <button onclick="window.uiController.clearExpandedView()"
                    class="btn-icon hover:bg-space-700 rounded">
              <span class="material-icons text-sm">close</span>
            </button>
          </div>
        </div>
        <div class="p-4">
          <div class="mb-4">
            <div id="planet-type-size" class="text-sm text-space-300 mb-3"></div>
            <div id="planet-resources-icons" class="flex gap-2 justify-center p-3 bg-gradient-to-r from-space-800/50 to-space-700/50 rounded-lg"></div>
          </div>

          <div id="planet-details-scroll-container" class="flex-1 overflow-y-auto pr-2 custom-scrollbar space-y-4">
            <div id="planet-resources-container" style="display: none;">
              <h3 class="text-lg font-semibold mb-3 text-nebula-200">Resources & Stats</h3>
              <div id="planet-resources-html"></div>
            </div>
            <div id="planet-buildings-container" style="display: none;">
              <h3 class="text-lg font-semibold mb-3 text-nebula-200">Buildings</h3>
              <div id="planet-buildings-html"></div>
            </div>
          </div>

          <div id="planet-actions-container" class="mt-4 space-y-2">
            <!-- Action buttons will be dynamically added here -->
          </div>
        </div>
      </div>
      `;const y=this.getPlanetTypeGradient(a),x=i.querySelector("#planet-header");x.className=`panel-header flex justify-between items-center p-3 cursor-move border-b border-space-700/50 bg-gradient-to-r ${y}`,i.querySelector("#planet-icon").innerHTML=this.getPlanetAnimatedGif(l)||r,i.querySelector("#planet-name").textContent=o,i.querySelector("#planet-seed").textContent=`Seed: ${e.id.slice(-8)}`,i.querySelector("#planet-system").textContent=c,i.querySelector("#planet-type-size").innerHTML=`
      <div class="text-center">
        <div class="font-medium">${l}</div>
        <div class="text-xs text-space-400">Size ${e.size||"N/A"} • ${c}</div>
      </div>
    `;const S=await this.getResourceIcons(e);i.querySelector("#planet-resources-icons").innerHTML=S.join("");let C='<div class="text-sm text-space-400">No resource deposits detected.</div>';if(n&&n.length>0){const N=await this.loadResourceTypes(),Y={};N.forEach(F=>{Y[F.id]=F});const B={};n.forEach(F=>{let _,R;F.expand&&F.expand.resource_type?(_=F.expand.resource_type.name,R=F.expand.resource_type):(R=Y[F.resource_type],_=R?R.name:F.resource_type),_&&(B[_]||(B[_]={nodes:[],resourceTypeData:R}),B[_].nodes.push(F))}),C=`<ul class="space-y-2">${Object.entries(B).map(([F,_])=>{const{nodes:R,resourceTypeData:z}=_,W=R.reduce((J,X)=>J+X.richness,0),V=(W/R.length).toFixed(1),G=R.length;return`
          <li class="p-3 bg-space-700 rounded-md flex items-center justify-between hover:bg-space-600 transition-colors">
            <div class="flex items-center gap-3">
              <img src="${(z==null?void 0:z.icon)||"/placeholder-planet.svg"}" class="w-6 h-6" title="${F}" alt="${F}" />
              <div>
                <span class="font-semibold">${F}</span>
                <div class="text-xs text-space-400">${G} deposit${G>1?"s":""}</div>
              </div>
            </div>
            <div class="text-right">
              <div class="text-sm font-medium">Richness: ${V}</div>
              <div class="text-xs text-space-400">Total: ${W}</div>
            </div>
          </li>
        `}).join("")}</ul>`}const k=i.querySelector("#planet-resources-container"),P=i.querySelector("#planet-buildings-container");i.querySelector("#planet-resources-html").innerHTML=C,k.style.display="block";const U=k.querySelector("h3");U&&(U.textContent="Resource Deposits"),v?(i.querySelector("#planet-buildings-html").innerHTML=u,P.style.display="block"):P.style.display="none";const I=i.querySelector("#planet-actions-container");if(I.innerHTML="",g){if(this.getAvailableSettlerFleets(e.system_id).length>0){const N=document.createElement("button");N.className="w-full btn btn-success py-3 flex items-center justify-center gap-2",N.innerHTML='<span class="material-icons">rocket_launch</span> Colonize Planet (Settler Ship)',N.onclick=()=>window.uiController.colonizePlanetWrapper(e.id),I.appendChild(N)}}else if(!f&&this.currentUser&&this.gameState){const E=document.createElement("button");E.className="w-full btn btn-disabled py-3 flex items-center justify-center gap-2",E.innerHTML='<span class="material-icons">rocket_launch</span> Colonize Planet (Need Settler Ship)',E.disabled=!0,I.appendChild(E)}if(v){const E=document.createElement("button");E.className="w-full btn btn-primary py-3 flex items-center justify-center gap-2",E.innerHTML="<span>🏗️</span> Construct Building",E.onclick=()=>window.uiController.showPlanetBuildModal(e),I.appendChild(E)}const A=document.createElement("button");A.className="w-full btn btn-secondary py-3 flex items-center justify-center gap-2",A.textContent="← Back to System",A.onclick=()=>window.uiController.goBackToSystemView(e.system_id),I.appendChild(A),i.classList.remove("hidden"),t!==void 0&&s!==void 0?this.positionPanel(i,t,s):(i.style.left==="-2000px"||i.style.left==="-9999px"||!i.style.left)&&(i.style.top="20px",i.style.left="20px",i.style.right="auto"),this.makePanelDraggable(i)}showPlanetBuildModal(e){var l,c,h;if(!this.currentUser){this.showError("Please log in to construct buildings.");return}if(!e||!e.id){this.showError("Invalid planet data provided for construction.");return}const t=(l=this.gameState)==null?void 0:l.buildingTypes;if(!t||t.length===0){console.warn("Building types not available or empty in gameState for showPlanetBuildModal."),this.showModal(`Construct on ${e.name||`Planet ${e.id.slice(-4)}`}`,`<div class="text-space-400">No building types available or data is still loading.</div>
         <button class="w-full mt-2 btn btn-secondary" onclick="window.uiController.hideModal()">Close</button>`);return}const s=new Set;e.resourceNodes&&e.resourceNodes.length>0&&e.resourceNodes.forEach(d=>{d.expand&&d.expand.resource_type&&s.add(d.expand.resource_type.name.toLowerCase())});const i=e.system_id,n=this.currentUser,o=((h=(c=this.gameState)==null?void 0:c.fleets)==null?void 0:h.filter(d=>d.current_system===i&&d.owner_id===(n==null?void 0:n.id)))||[],a=t.map(d=>{var k;let u="Cost: ",p=!0,f=[];if(d.cost_resource_type&&d.cost_quantity>0){const P=d.cost_resource_name||"Unknown Resource";u+=`${d.cost_quantity} ${P}`;let U=!1;o.length>0&&(U=!0),U||(p=!1,f.push(`${d.cost_quantity} ${P}`))}else d.cost>0?u+=`${d.cost} Credits`:u+="Free";if(d.resource_nodes&&d.resource_nodes.length>0){const P=d.resource_nodes,U=(((k=this.gameState)==null?void 0:k.resourceTypes)||[]).reduce((I,A)=>(I[A.id]=A.name,I),{});P.forEach(I=>{const A=U[I]||I;s.has(A.toLowerCase())||(p=!1,f.push(A))})}const g=e.id.replace(/'/g,"\\'"),y=d.id.replace(/'/g,"\\'"),x=p?"w-full p-3 bg-space-700 hover:bg-space-600 rounded mb-2 text-left cursor-pointer":"w-full p-3 bg-space-800 rounded mb-2 text-left cursor-not-allowed opacity-60",S=p&&o.length>0?`onclick="window.gameState.queueBuilding('${g}', '${y}', '${o[0].id}'); window.uiController.hideModal();"`:"";o.length===0&&(p=!1,f.push("Fleet at this system"));let C="";return!p&&f.length>0&&(C=`<div class="text-xs text-red-400 mt-1">Missing: ${f.join(", ")}</div>`),`
      <button class="${x}" ${S} ${p?"":"disabled"}>
        <div class="font-semibold ${p?"":"text-space-400"}">${d.name||"Unknown Building"}</div>
        <div class="text-sm text-space-300">${d.description||"No description available."}</div>
        <div class="text-sm">${u}</div>
        ${C}
      </button>
    `}).join(""),r=s.size>0?`<div class="mb-4 p-3 bg-space-800 rounded">
           <div class="text-sm font-semibold mb-2">Available Resources:</div>
           <div class="text-xs text-space-300">${Array.from(s).map(d=>d.charAt(0).toUpperCase()+d.slice(1)).join(", ")}</div>
         </div>`:`<div class="mb-4 p-3 bg-space-800 rounded">
           <div class="text-sm text-red-400">No resource deposits found on this planet.</div>
         </div>`;this.showModal(`Construct on ${e.name||`Planet ${e.id.slice(-4)}`}`,`
      ${r}
      <div class="space-y-2 max-h-96 overflow-y-auto">
        ${a.length>0?a:'<div class="text-space-400">No buildings available to construct.</div>'}
      </div>
      <button class="w-full mt-4 btn btn-secondary" onclick="window.uiController.hideModal()">Cancel</button>
    `)}getAvailableSettlerFleets(e){return!this.gameState||!this.gameState.fleets?[]:this.gameState.fleets.filter(t=>{var s;return t.current_system!==e||t.owner_id!==((s=this.currentUser)==null?void 0:s.id)?!1:t.ships&&t.ships.some(i=>i.ship_type_name==="settler"&&i.count>0)})}colonizePlanetWrapper(e){if(!this.gameState||!this.gameState.mapData||!this.gameState.mapData.planets){this.showError("Game data not loaded. Cannot colonize.");return}const t=this.gameState.mapData.planets.find(s=>s.id===e);if(!t){this.showError("Planet data not found. Cannot colonize.");return}this.colonizePlanet(t.id,t.system_id)}async colonizePlanet(e,t=null){var s,i,n;try{const{pb:o}=await be(async()=>{const{pb:d}=await Promise.resolve().then(()=>Ve);return{pb:d}},void 0);if(!o.authStore.isValid){this.showError("Please log in first to colonize planets");return}let a=t;if(!a){const d=(n=(i=(s=this.gameState)==null?void 0:s.mapData)==null?void 0:i.planets)==null?void 0:n.find(u=>u.id===e);if(!d){this.showError("Planet not found in game data");return}a=d.system_id}const r=this.getAvailableSettlerFleets(a);if(r.length===0){this.showError("No settler ships available at this system");return}const l=r[0],c=await fetch(`${o.baseUrl}/api/orders/colonize`,{method:"POST",headers:{"Content-Type":"application/json",Authorization:o.authStore.token},body:JSON.stringify({planet_id:e,fleet_id:l.id})}),h=await c.json();if(c.ok&&h.success){this.hideModal();const{gameState:d}=await be(async()=>{const{gameState:u}=await Promise.resolve().then(()=>He);return{gameState:u}},void 0);await d.refreshGameData(),this.gameState&&this.gameState.selectedSystem&&this.gameState.selectedSystem.id===a&&setTimeout(()=>{this.displaySystemView(this.gameState.selectedSystem)},100),this.showSuccessMessage("Planet colonized successfully! Your settler ship has established a new colony.")}else throw new Error(h.message||"Colonization failed")}catch(o){console.error("Colonization error:",o),this.showError(`Failed to colonize planet: ${o.message}`)}}goBackToSystemView(){const e=this.gameState.getSelectedSystem();if(e){let t=[];this.gameState.mapData.planets&&(t=this.gameState.mapData.planets.filter(s=>Array.isArray(s.system_id)?s.system_id.includes(e.id):s.system_id===e.id)),this.displaySystemView(e,t)}else this.clearExpandedView()}async displayFleetView(e,t,s){const i=this.expandedView;if(!i){console.error("#expanded-view-container not found in displayFleetView");return}const n=i._isPinned;i.classList.contains("pinned-panel"),n||this.clearExpandedView(),i.classList.remove("hidden"),i.classList.add("floating-panel","glass-panel"),n&&(i.classList.add("pinned-panel"),i._isPinned=!0);const o=this.renderFleetView(e),a=e.name||`Fleet ${e.id.slice(-4)}`;i.innerHTML=`
      <div class="panel-header flex items-center justify-between p-3 cursor-move border-b border-space-600/30">
        <div class="flex items-center gap-2">
          <span class="material-icons drag-handle text-space-400">drag_indicator</span>
          <h2 class="text-lg font-semibold text-white">${a}</h2>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="window.uiController.togglePinPanel(this.closest('.floating-panel'))" 
                  class="pin-button text-space-400 hover:text-white transition-colors" 
                  title="Pin panel">
            <span class="material-icons text-sm">push_pin</span>
          </button>
          <button onclick="window.uiController.clearExpandedView()" 
                  class="text-space-400 hover:text-white transition-colors"
                  title="Close panel">
            <span class="material-icons">close</span>
          </button>
        </div>
      </div>
      <div class="floating-panel-content p-4">
        ${o}
      </div>
    `,t!==null&&s!==null&&!i._isPinned&&this.positionPanel(i,t,s),this.setupPanelDragging(i),this.addPanelFocusEffects(i)}setupPanelDragging(e){this.makePanelDraggable(e)}addPanelFocusEffects(e){e.addEventListener("mousedown",()=>{document.querySelectorAll(".floating-panel.focused").forEach(t=>{t!==e&&t.classList.remove("focused")}),e.classList.add("focused")}),document.addEventListener("click",t=>{!e.contains(t.target)&&!e._isPinned&&e.classList.remove("focused")})}togglePinPanel(e){if(!e)return;const t=e.classList.contains("pinned-panel"),s=e.querySelector(".pin-button"),i=s==null?void 0:s.querySelector(".material-icons");t?(e.classList.remove("pinned-panel"),e._isPinned=!1,s&&(s.classList.remove("pinned"),s.title="Pin panel"),i&&(i.textContent="push_pin",i.style.transform=""),this.showToast("Panel unpinned","info",1500)):(e.classList.add("pinned-panel"),e._isPinned=!0,s&&(s.classList.add("pinned"),s.title="Unpin panel - will stay in current position"),i&&(i.textContent="push_pin"),this.showToast("Panel pinned - will stay anchored here","success",2e3))}renderFleetView(e){return this.fleetComponentManager?this.fleetComponentManager.fleetComponent.render(e.id):`
        <div class="p-4 bg-space-700 rounded-lg border border-space-600">
          <h3 class="text-red-400 font-semibold mb-2">Fleet System Not Available</h3>
          <p class="text-space-300 text-sm">Fleet management system is not initialized.</p>
        </div>
      `}manageColony(e){if(!this.gameState||!this.gameState.mapData||!this.gameState.mapData.planets){this.showError("Game data not loaded. Cannot manage colony.");return}const t=this.gameState.mapData.planets.find(i=>i.id===e);if(!t){this.showError("Planet data not found.");return}const s=this.gameState.mapData.systems.find(i=>i.id===t.system_id);s?this.showBuildModal(s):this.showError("System for this planet not found.")}updateAuthUI(e){this.currentUser=e;const t=document.getElementById("login-btn"),s=document.getElementById("user-info"),i=document.getElementById("username");e?(t.classList.add("hidden"),s.classList.remove("hidden"),i.textContent=e.username):(t.classList.remove("hidden"),s.classList.add("hidden"),i.textContent="")}updateGameUI(e){this.gameState=e,!this.fleetComponentManager&&this.currentUser?this.fleetComponentManager=new Qe(this,e):this.fleetComponentManager&&this.fleetComponentManager.updateGameState(e),this.updateResourcesUI(e.playerResources),this.updateGameStatusUI(e)}loadResourcePreferences(){try{const e=localStorage.getItem("xan_displayed_resources");return e?JSON.parse(e):["credits","ore"]}catch{return["credits","ore"]}}saveResourcePreferences(){try{localStorage.setItem("xan_displayed_resources",JSON.stringify(this.displayedResources))}catch(e){console.warn("Failed to save resource preferences:",e)}}initializeResourcesDropdown(){document.addEventListener("click",e=>{const t=document.getElementById("resources-dropdown"),s=document.getElementById("resources-toggle");if(e.target.closest("#resources-toggle"))if(t.classList.contains("hidden")){const i=s.getBoundingClientRect(),n=document.getElementById("game-canvas").getBoundingClientRect(),o=i.right-n.left-256,a=i.bottom-n.top+8;t.style.left=`${o}px`,t.style.top=`${a}px`,t.classList.remove("hidden")}else t.classList.add("hidden");else e.target.closest("#resources-dropdown")||t.classList.add("hidden")}),document.addEventListener("click",e=>{e.target.closest("#resources-settings")&&this.showResourcesSettingsModal()})}updateResourcesDropdown(){var i,n;const e=document.getElementById("resources-list");if(!e)return;const t=((i=this.gameState)==null?void 0:i.playerResources)||{};e.innerHTML="";const s=[{name:"credits",icon:"account_balance_wallet",color:"text-nebula-300"},{name:"ore",icon:"construction",color:"text-orange-400"},{name:"food",icon:"restaurant",color:"text-green-400"},{name:"fuel",icon:"local_gas_station",color:"text-blue-400"},{name:"metal",icon:"build",color:"text-gray-400"},{name:"titanium",icon:"precision_manufacturing",color:"text-purple-400"},{name:"xanium",icon:"auto_awesome",color:"text-yellow-400"}];for(const o of s){const a=t[o.name]||0;if(o.name==="credits"){const r=((n=this.gameState)==null?void 0:n.creditIncome)>0?` (+${this.gameState.creditIncome}/tick)`:"";e.innerHTML+=`
          <div class="flex items-center justify-between py-1">
            <div class="flex items-center gap-2">
              <span class="material-icons text-sm ${o.color}">${o.icon}</span>
              <span class="text-sm">Credits</span>
            </div>
            <div class="text-sm font-mono">
              <span class="${o.color}">${a.toLocaleString()}</span>
              ${r?`<span class="text-xs text-plasma-300">${r}</span>`:""}
            </div>
          </div>
        `}else{const r=this.resourceTypes.get(o.name.toLowerCase()),l=r!=null&&r.icon?`<img src="${r.icon}" class="w-4 h-4" alt="${o.name}" />`:`<span class="material-icons text-sm ${o.color}">${o.icon}</span>`;e.innerHTML+=`
          <div class="flex items-center justify-between py-1">
            <div class="flex items-center gap-2">
              ${l}
              <span class="text-sm capitalize">${o.name}</span>
            </div>
            <span class="text-sm font-mono text-space-200">${a.toLocaleString()}</span>
          </div>
        `}}}updateResourcesUI(e){var s;const t=document.getElementById("resources-display");if(t){t.innerHTML="";for(const i of this.displayedResources){const n=e[i];if(n!==void 0)if(i==="credits"){t.innerHTML+=`
          <button id="credits-btn" class="text-nebula-300 hover:text-nebula-200 hover:bg-nebula-900/20 px-2 py-1 rounded transition-all cursor-pointer border border-nebula-600/30 hover:border-nebula-500/50 flex items-center gap-1">
            <span class="material-icons text-base">account_balance_wallet</span>
            <span id="credits">${(n==null?void 0:n.toLocaleString())||0}</span><span id="credit-income" class="text-xs text-plasma-300 ml-1"></span>
          </button>
        `;const o=document.getElementById("credit-income");o&&(((s=this.gameState)==null?void 0:s.creditIncome)>0?(o.textContent=`(+${this.gameState.creditIncome}/tick)`,o.style.display="inline"):o.style.display="none");const a=document.getElementById("credits-btn");a&&(a.onclick=()=>this.showCreditsBreakdown())}else{const o=this.getResourceDefinition(i),a=this.resourceTypes.get(i.toLowerCase()),r=a!=null&&a.icon?`<img src="${a.icon}" class="w-4 h-4" alt="${i}" />`:`<span class="material-icons text-base">${o.icon}</span>`,l=o.color||"text-space-300";t.innerHTML+=`
          <button class="${l} hover:text-space-200 hover:bg-space-900/20 px-2 py-1 rounded transition-all cursor-pointer border border-space-600/30 hover:border-space-500/50 flex items-center gap-1">
            ${r}
            <span class="font-mono">${(n==null?void 0:n.toLocaleString())||0}</span>
          </button>
        `}}this.updateResourcesDropdown()}}getResourceDefinition(e){return{credits:{icon:"account_balance_wallet",color:"text-nebula-300"},ore:{icon:"construction",color:"text-orange-400"},food:{icon:"restaurant",color:"text-green-400"},fuel:{icon:"local_gas_station",color:"text-blue-400"},metal:{icon:"build",color:"text-gray-400"},titanium:{icon:"precision_manufacturing",color:"text-purple-400"},xanium:{icon:"auto_awesome",color:"text-yellow-400"}}[e]||{icon:"inventory",color:"text-space-300"}}showResourcesSettingsModal(){const t=[{name:"credits",icon:"account_balance_wallet"},{name:"ore",icon:"construction"},{name:"food",icon:"restaurant"},{name:"fuel",icon:"local_gas_station"},{name:"metal",icon:"build"},{name:"titanium",icon:"precision_manufacturing"},{name:"xanium",icon:"auto_awesome"}].map(s=>{const i=this.displayedResources.includes(s.name),n=this.resourceTypes.get(s.name),o=n!=null&&n.icon?`<img src="${n.icon}" class="w-4 h-4" alt="${s.name}" />`:`<span class="material-icons text-sm">${s.icon}</span>`;return`
        <label class="flex items-center gap-3 p-2 hover:bg-space-700/50 rounded cursor-pointer">
          <input type="checkbox" ${i?"checked":""}
                 class="resource-checkbox" data-resource="${s.name}">
          <div class="flex items-center gap-2">
            ${o}
            <span class="capitalize">${s.name}</span>
          </div>
        </label>
      `}).join("");this.showModal("Resource Display Settings",`
      <form id="resources-settings-form" class="space-y-4">
        <div>
          <p class="text-sm text-space-400 mb-3">Choose which resources to display in the top bar:</p>
          <div class="space-y-1 max-h-60 overflow-y-auto">
            ${t}
          </div>
        </div>
        <div class="flex space-x-2">
          <button type="submit" class="flex-1 btn btn-success">
            Save Settings
          </button>
          <button type="button" onclick="document.getElementById('modal-overlay').classList.add('hidden')"
                  class="flex-1 btn btn-secondary">
            Cancel
          </button>
        </div>
      </form>
      `),document.getElementById("resources-settings-form").addEventListener("submit",s=>{var n;s.preventDefault();const i=document.querySelectorAll(".resource-checkbox");this.displayedResources=Array.from(i).filter(o=>o.checked).map(o=>o.dataset.resource),this.displayedResources.length===0&&(this.displayedResources=["credits"]),this.saveResourcePreferences(),this.updateResourcesUI(((n=this.gameState)==null?void 0:n.playerResources)||{}),this.hideModal()})}updateGameStatusUI(e){const t=document.getElementById("game-tick-display");if(t){const i=t.textContent,n=`Tick: ${e.currentTick}`;t.textContent=n,i!==n&&i!=="Tick: 0"&&(t.style.animation="none",t.offsetHeight,t.style.animation="flash 0.5s ease-out")}const s=document.getElementById("next-tick-display");if(s&&!this.tickTimer){const i=e.ticksPerMinute||6,n=Math.round(60/i);s.textContent=`Next Tick: (${n}s period)`}}startTickTimer(e){this.tickTimer&&clearInterval(this.tickTimer);const t=document.getElementById("next-tick-display"),s=()=>{const n=e-new Date;if(n<=0){t&&(t.textContent="Next Tick: Processing..."),clearInterval(this.tickTimer),this.tickTimer=null;return}const o=Math.floor(n/6e4),a=Math.floor(n%6e4/1e3);t&&(t.textContent=`Next Tick: ${o}:${a.toString().padStart(2,"0")}`)};s(),this.tickTimer=setInterval(s,1e3)}showModal(e,t){const s=document.getElementById("modal-overlay"),i=document.getElementById("modal-content");i.classList.add("modal-content"),i.innerHTML=`
      <div class="modal-header panel-header cursor-move flex justify-between items-center p-4">
        <div class="flex items-center gap-2">
          <span class="material-icons drag-handle text-space-400">drag_indicator</span>
          <h2 class="text-xl font-bold">${e}</h2>
        </div>
        <div class="flex items-center gap-2">
          <button onclick="window.uiController.togglePinModal()" 
                  class="pin-button text-space-400 hover:text-white transition-colors" 
                  title="Pin modal">
            <span class="material-icons text-sm">push_pin</span>
          </button>
          <button id="modal-close" class="text-space-400 hover:text-white transition-colors" title="Close modal">
            <span class="material-icons">close</span>
          </button>
        </div>
      </div>
      <div class="modal-body p-4">
        ${t}
      </div>
    `,s.classList.remove("hidden"),document.getElementById("modal-close").addEventListener("click",()=>{this.hideModal()}),this.makePanelDraggable(i),this.addPanelFocusEffects(i),i.style.left="",i.style.top="",i.style.transform="translate(-50%, -50%)"}hideModal(){const e=document.getElementById("modal-overlay"),t=document.getElementById("modal-content");t._isPinned||(t._dragCleanup&&(t._dragCleanup(),delete t._dragCleanup),e.classList.add("hidden"),t.classList.remove("focused","pinned-panel"),t._isPinned=!1)}togglePinModal(){const e=document.getElementById("modal-content");this.togglePinPanel(e)}showError(e){this.showModal("Error",`
      <div class="text-red-400 mb-4">${e}</div>
      <button class="w-full btn btn-secondary" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
        OK
      </button>
    `)}showBuildModal(e){var i;const t=(i=this.gameState)==null?void 0:i.buildingTypes;if(!t||t.length===0){console.warn("Building types not available or empty in gameState."),this.showModal(`Build in ${e.name||`System ${e.id.slice(-3)}`}`,'<div class="text-space-400">No buildings available to construct or building types are still loading.</div>');return}const s=t.map(n=>{var a;let o="Cost: ";if(typeof n.cost=="number")o+=`${n.cost} Credits`;else if(typeof n.cost=="object"){const r=(((a=this.gameState)==null?void 0:a.resourceTypes)||[]).reduce((l,c)=>(l[c.id]=c.name,l),{});o+=Object.entries(n.cost).map(([l,c])=>{const h=r[l]||l;return`${c} ${h}`}).join(", ")}else o+="N/A";return`
      <button class="w-full p-3 bg-space-700 hover:bg-space-600 rounded mb-2 text-left"
              onclick="window.gameState.queueBuilding('${e.id}', '${n.id}')">
        <div class="font-semibold">${n.name||"Unknown Building"}</div>
        <div class="text-sm text-space-300">${n.description||"No description available."}</div>
        <div class="text-sm text-green-400">${o}</div>
      </button>
    `}).join("");this.showModal(`Build in ${e.name||`System ${e.id.slice(-3)}`}`,`
      <div class="space-y-2">
        ${s.length>0?s:'<div class="text-space-400">No buildings available to construct.</div>'}
      </div>
    `)}showSendFleetModal(e){var i;const t=((i=this.gameState)==null?void 0:i.getOwnedSystems())||[];if(t.length===0){this.showError("You need to own at least one system to send fleets");return}const s=t.map(n=>`<option value="${n.id}">${n.name||`System ${n.id.slice(-3)}`}</option>`).join("");this.showModal("Send Fleet",`
      <form id="fleet-form" class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">From System:</label>
          <select id="from-system" class="w-full p-2 bg-space-700 border border-space-600 rounded">
            ${s}
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">To System:</label>
          <input type="text" id="to-system" value="${e.name||`System ${e.id.slice(-3)}`}"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded" readonly>
          <input type="hidden" id="to-system-id" value="${e.id}">
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Fleet Strength:</label>
          <input type="number" id="fleet-strength" min="1" max="100" value="10"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded">
        </div>
        <div class="flex space-x-2">
          <button type="submit" class="flex-1 btn btn-danger">
            Send Fleet
          </button>
          <button type="button" onclick="document.getElementById('modal-overlay').classList.add('hidden')"
                  class="flex-1 btn btn-secondary">
            Cancel
          </button>
        </div>
      </form>
    `),document.getElementById("fleet-form").addEventListener("submit",async n=>{n.preventDefault();try{const o=document.getElementById("from-system").value,a=document.getElementById("to-system-id").value,r=parseInt(document.getElementById("fleet-strength").value);await this.gameState.sendFleet(o,a,r),this.hideModal()}catch(o){this.showError(`Failed to send fleet: ${o.message}`)}})}showTradeRouteModal(e){var o;const t=((o=this.gameState)==null?void 0:o.getOwnedSystems())||[];if(t.length===0){this.showError("You need to own at least one system to create trade routes");return}const s=t.map(a=>`<option value="${a.id}">${a.name||`System ${a.id.slice(-3)}`}</option>`).join(""),n=["food","ore","goods","fuel"].map(a=>`<option value="${a}">${a.charAt(0).toUpperCase()+a.slice(1)}</option>`).join("");this.showModal("Create Trade Route",`
      <form id="trade-form" class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">From System:</label>
          <select id="trade-from-system" class="w-full p-2 bg-space-700 border border-space-600 rounded">
            ${s}
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">To System:</label>
          <input type="text" value="${e.name||`System ${e.id.slice(-3)}`}"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded" readonly>
          <input type="hidden" id="trade-to-system-id" value="${e.id}">
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Cargo Type:</label>
          <select id="cargo-type" class="w-full p-2 bg-space-700 border border-space-600 rounded">
            ${n}
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Cargo Capacity:</label>
          <input type="number" id="cargo-capacity" min="1" max="1000" value="100"
                 class="w-full p-2 bg-space-700 border border-space-600 rounded">
        </div>
        <div class="flex space-x-2">
          <button type="submit" class="flex-1 btn btn-success">
            Create Route
          </button>
          <button type="button" onclick="document.getElementById('modal-overlay').classList.add('hidden')"
                  class="flex-1 btn btn-secondary">
            Cancel
          </button>
        </div>
      </form>
    `),document.getElementById("trade-form").addEventListener("submit",async a=>{a.preventDefault();try{const r=document.getElementById("trade-from-system").value,l=document.getElementById("trade-to-system-id").value,c=document.getElementById("cargo-type").value,h=parseInt(document.getElementById("cargo-capacity").value);await this.gameState.createTradeRoute(r,l,c,h),this.hideModal()}catch(r){this.showError(`Failed to create trade route: ${r.message}`)}})}showFleetPanel(){if(!this.fleetComponentManager){this.showModal("Your Fleets",'<div class="text-space-400">Fleet system not initialized.</div>');return}this.fleetComponentManager.showFleetPanel()}showTradePanel(){var s;const e=((s=this.gameState)==null?void 0:s.getPlayerTrades())||[],t=e.length>0?e.map(i=>`
      <div class="bg-space-700 p-3 rounded mb-2">
        <div class="font-semibold">Trade Route ${i.id.slice(-3)}</div>
        <div class="text-sm text-space-300">
          <div>From: ${i.from_name||i.from_id}</div>
          <div>To: ${i.to_name||i.to_id}</div>
          <div>Cargo: ${i.cargo}</div>
          <div>Capacity: ${i.cap}</div>
          <div>ETA: ${i.eta_tick?`Tick ${i.eta_tick}`:"Unknown"}</div>
        </div>
      </div>
    `).join(""):'<div class="text-space-400">No active trade routes</div>';this.showModal("Your Trade Routes",t)}showDiplomacyPanel(){this.showModal("Diplomacy",`
      <div class="text-center text-space-400 py-8">
        Diplomacy features coming soon!
      </div>
    `)}showBuildingsPanel(){var a;const e=((a=this.gameState)==null?void 0:a.getPlayerBuildings())||[],s=e.filter(r=>r.credits_per_tick>0).reduce((r,l)=>r+l.credits_per_tick,0),i=e.reduce((r,l)=>(r[l.type]||(r[l.type]=[]),r[l.type].push(l),r),{}),n={};if(this.gameState&&this.gameState.buildingTypes)for(const r of this.gameState.buildingTypes)n[r.id]=r.name||r.id;else console.warn("Building types not available in gameState for building panel.");const o=Object.entries(i).map(([r,l])=>`
      <div class="mb-4">
        <h3 class="text-lg font-semibold text-plasma-300 mb-2">${n[r]||r} (${l.length})</h3>
        <div class="space-y-2">
          ${l.map(c=>`
            <div class="bg-space-700 p-3 rounded">
              <div class="font-semibold text-nebula-300">${c.name||`${n[c.type]||c.type} ${c.id.slice(-3)}`}</div>
              <div class="text-sm text-space-300">
                <div>System: ${c.system_name||c.system_id}</div>
                ${c.credits_per_tick>0?`<div class="text-nebula-300">Income: ${c.credits_per_tick} credits/tick</div>`:""}
                <div class="text-xs ${c.active!==!1?"text-green-400":"text-red-400"}">
                  ${c.active!==!1?"Active":"Inactive"}
                </div>
              </div>
            </div>
          `).join("")}
        </div>
      </div>
    `).join("");this.showModal("Buildings Overview",`
      ${s>0?`
        <div class="mb-4 p-3 bg-space-800 rounded">
          <div class="text-lg font-semibold text-plasma-300">Credit Income: ${s} credits/tick</div>
          <div class="text-sm text-space-400">${s*6} credits/minute • ${s*360} credits/hour</div>
        </div>
      `:""}

      ${o||'<div class="text-space-400 text-center py-8">No buildings constructed</div>'}

      <div class="mt-4 text-xs text-space-400 border-t border-space-600 pt-2">
        💡 Build structures at your systems to improve production and defense
      </div>
    `)}showColonizeModal(e){if(!this.currentUser){this.showError("Please log in to colonize planets");return}fetch(`http://localhost:8090/api/planets?system_id=${e.id}`).then(t=>t.json()).then(t=>{const s=t.items||[];if(s.length===0){this.showError("No planets found in this system");return}const i=s.map(n=>{const o=n.colonized_by!=null&&n.colonized_by!=="",a=this.getPlanetTypeName(n.type)||"Unknown";return`
            <div class="p-3 bg-space-700 rounded mb-2 ${o?"opacity-50":"hover:bg-space-600 cursor-pointer"}"
                 ${o?"":`onclick="window.uiController.colonizePlanet('${n.id}')"`}>
              <div class="font-semibold">${n.name}</div>
              <div class="text-sm text-space-300">Type: ${a}</div>
              <div class="text-sm text-space-300">Size: ${n.size}</div>
              ${o?'<div class="text-sm text-red-400">Already colonized</div>':'<div class="text-sm text-emerald-400">Available for colonization</div>'}
            </div>
          `}).join("");this.showModal(`Colonize Planet in ${e.name||`System ${e.id.slice(-3)}`}`,`
          <div class="space-y-2">
            <div class="text-sm text-space-300 mb-4">
              Select a planet to establish a new colony:
            </div>
            ${i}
          </div>
        `),window.uiController=this}).catch(t=>{console.error("Error fetching planets:",t),this.showError("Failed to load planets in this system")})}showToast(e,t="success",s=4e3,i=null){const n=document.getElementById("fleet-toast");n&&n.remove();const o=document.createElement("div");o.id="fleet-toast",o.className="fixed bottom-20 left-1/2 transform -translate-x-1/2 z-50 p-2 rounded shadow-md transition-all duration-200 max-w-xs",i&&(o.className+=" cursor-pointer hover:opacity-80"),t==="success"?o.className+=" bg-emerald-900/90 border-emerald-600 text-emerald-200":t==="error"?o.className+=" bg-red-900/90 border-red-600 text-red-200":t==="info"?o.className+=" bg-blue-900/90 border-blue-600 text-blue-200":t==="ticket"&&(o.className+=" bg-slate-900/95 text-slate-200",o.style.maxWidth="280px");const a=()=>{o.parentElement&&o.remove(),document.removeEventListener("keydown",r)};if(t==="ticket")o.innerHTML=`
        <div class="flex items-start justify-between">
          <div class="flex-1">${e}</div>
          <button onclick="event.stopPropagation(); this.parentElement.parentElement.remove()" class="ml-2 text-current opacity-50 hover:opacity-100">
            <span class="material-icons text-xs">close</span>
          </button>
        </div>
      `;else{const l=i?'<div class="text-xs mt-1 opacity-70">Click for details • Press Space or Esc to dismiss</div>':'<div class="text-xs mt-1 opacity-70">Press Space or Esc to dismiss</div>';o.innerHTML=`
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="material-icons text-sm">${t==="success"?"check_circle":t==="error"?"error":"info"}</span>
            <span class="text-sm">${e}</span>
          </div>
          <button onclick="event.stopPropagation(); this.parentElement.parentElement.remove()" class="ml-2 text-current opacity-70 hover:opacity-100">
            <span class="material-icons text-sm">close</span>
          </button>
        </div>
        ${l}
      `}i&&o.addEventListener("click",l=>{l.target.closest("button")||(i(),a())}),document.body.appendChild(o),s>0&&setTimeout(()=>{a()},s);const r=l=>{(l.code==="Space"||l.code==="Escape")&&(l.preventDefault(),a())};document.addEventListener("keydown",r)}showSuccessMessage(e){this.showToast(e,"success")}showCreditsBreakdown(){var o,a,r;if(!this.currentUser){this.showError("Please log in to view credit breakdown");return}const t=(((o=this.gameState)==null?void 0:o.getPlayerBuildings())||[]).filter(l=>{var h,d,u;return((u=(d=(h=this.gameState)==null?void 0:h.buildingTypes)==null?void 0:d.find(p=>p.id===l.type))==null?void 0:u.name)==="crypto_server"});let s=((r=(a=this.gameState)==null?void 0:a.playerResources)==null?void 0:r.credits)||0,i=0;t.forEach(l=>{l.credits_per_tick&&(i+=l.credits_per_tick)});const n=t.length>0?t.map(l=>{var d;const c=l.system_name||`System ${(d=l.system_id)==null?void 0:d.slice(-3)}`;l.stored_credits;const h=l.credits_per_tick||1;return`
            <div class="bg-space-700 p-3 rounded mb-2">
              <div class="flex justify-between items-center">
                <div>
                  <div class="font-semibold text-nebula-300">Crypto Server</div>
                  <div class="text-sm text-space-300">Location: ${c}</div>
                </div>
                <div class="text-right">
                  <div class="text-nebula-300">+${h}/tick</div>
                  <div class="text-xs text-space-400">Level ${l.level||1}</div>
                </div>
              </div>
            </div>
          `}).join(""):'<div class="text-space-400 text-center py-4">No crypto servers found</div>';this.showModal('<span class="flex items-center gap-2"><span class="material-icons">account_balance_wallet</span>Credits Breakdown</span>',`
          <div class="space-y-4">
            <div class="bg-space-800 p-4 rounded-lg">
              <div class="grid grid-cols-2 gap-4 text-center">
                <div>
                  <div class="text-2xl font-bold text-nebula-300">${s.toLocaleString()}</div>
                  <div class="text-sm text-space-400">Total Credits</div>
                </div>
                <div>
                  <div class="text-2xl font-bold text-plasma-300">+${i}</div>
                  <div class="text-sm text-space-400">Per Tick</div>
                </div>
              </div>
            </div>

            <div>
              <h3 class="text-lg font-semibold mb-3 text-nebula-200">Credit Sources</h3>
              <div class="max-h-60 overflow-y-auto custom-scrollbar">
                ${n}
              </div>
            </div>

            ${t.length===0?`
              <div class="bg-amber-900/20 border border-amber-600/30 p-3 rounded">
                <div class="text-amber-300 text-sm">
                  💡 <strong>Tip:</strong> Build Crypto Servers on your planets to generate credits over time!
                </div>
              </div>
            `:""}
          </div>

          <button class="w-full btn btn-secondary mt-4" onclick="document.getElementById('modal-overlay').classList.add('hidden')">
            Close
          </button>
        `)}}const tt=new et;window.gameState=T;window.uiController=tt;class st{constructor(){this.mapRenderer=null,this.fleetRoutes=new Map,this.init()}async init(){console.log("Initializing Xan Nation..."),this.uiController=window.uiController,this.uiController.setPocketBase(b),this.mapRenderer=new We("game-canvas");const e=$.getUser();e&&this.mapRenderer.setCurrentUserId(e.id),$.subscribe(t=>{this.handleAuthChange(t)}),T.subscribe(t=>{this.handleGameStateChange(t)}),this.setupEventListeners(),console.log("Xandaris initialized")}handleAuthChange(e){this.uiController.updateAuthUI(e),this.mapRenderer&&this.mapRenderer.setCurrentUserId((e==null?void 0:e.id)||null),e?console.log("User logged in:",e.username):console.log("User logged out")}handleGameStateChange(e){var t,s;e.fleetOrders&&e.fleetOrders.length>0&&console.log(`Fleet orders updated: ${e.fleetOrders.length} orders`,e.fleetOrders),this.mapRenderer&&(this.mapRenderer.setSystems(e.systems),this.mapRenderer.setFleets(e.fleets),this.mapRenderer.setTrades(e.trades),this.mapRenderer.setHyperlanes(e.hyperlanes),((t=this.mapRenderer.selectedSystem)==null?void 0:t.id)!==((s=e.selectedSystem)==null?void 0:s.id)&&this.mapRenderer.setSelectedSystem(e.selectedSystem),e.mapData&&e.mapData.lanes&&this.mapRenderer.setLanes(e.mapData.lanes),e.centerOnFleetSystem&&!this.mapRenderer.hasCenteredOnFleet?(this.mapRenderer.centerOnSystem(e.centerOnFleetSystem),this.mapRenderer.hasCenteredOnFleet=!0,this.mapRenderer.zoom=.8):e.systems.length>0&&!this.mapRenderer.hasInitialFit&&(this.mapRenderer.fitToSystems(),this.mapRenderer.hasInitialFit=!0)),this.uiController.updateGameUI(e)}setupEventListeners(){const e=document.getElementById("game-canvas");e.addEventListener("systemSelected",i=>{const n=i.detail.system,o=i.detail.planets,a=i.detail.screenX,r=i.detail.screenY;(!T.selectedSystem||T.selectedSystem.id!==n.id)&&T.selectSystem(n.id),this.uiController.displaySystemView(n,o,a,r)}),e.addEventListener("fleetMoveRequested",i=>{const n=i.detail.fromFleet,o=i.detail.toSystem;this.handleMultiMoveFleet(n,o)}),e.addEventListener("fleetSelected",i=>{const n=i.detail.fleet,o=i.detail.screenX,a=i.detail.screenY;this.displaySelectedFleetInfo(n,o,a)});const t=document.getElementById("context-menu");t.addEventListener("click",i=>{const n=i.target.dataset.action,o=t.dataset.systemId;n&&o&&(this.handleContextMenuAction(n,o),t.classList.add("hidden"))}),e.addEventListener("mouseleave",()=>{document.getElementById("tooltip").classList.add("hidden")}),document.getElementById("fleet-btn").addEventListener("click",()=>{this.uiController.showFleetPanel()}),document.getElementById("trade-btn").addEventListener("click",()=>{this.uiController.showTradePanel()}),document.getElementById("diplo-btn").addEventListener("click",()=>{this.uiController.showDiplomacyPanel()}),document.getElementById("buildings-btn").addEventListener("click",()=>{this.uiController.showBuildingsPanel()}),document.getElementById("login-btn").addEventListener("click",()=>{this.handleLogin()}),document.getElementById("logout-btn").addEventListener("click",()=>{this.handleLogout()}),document.addEventListener("keydown",i=>{this.handleKeyboardInput(i)});const s=document.getElementById("modal-overlay");s.addEventListener("click",i=>{i.target===s&&this.uiController.hideModal()})}async handleLogin(){try{await $.loginWithDiscord()}catch(e){console.error("Login failed:",e),this.uiController.showError("Login failed. Please try again.")}}handleLogout(){$.logout()}handleContextMenuAction(e,t){const s=T.systems.find(i=>i.id===t);if(s)switch(e){case"view":T.selectSystem(t),this.mapRenderer.centerOnSystem(t);break;case"fleet":this.uiController.showSendFleetModal(s);break;case"trade":this.uiController.showTradeRouteModal(s);break}}handleBuildAction(){const e=T.getSelectedSystem();if(!e){this.uiController.showError("Please select a system first");return}if(!$.isLoggedIn()){this.uiController.showError("Please log in first");return}this.uiController.showBuildModal(e)}handleSendFleetAction(){const e=T.getSelectedSystem();if(!e){this.uiController.showError("Please select a system first");return}if(!$.isLoggedIn()){this.uiController.showError("Please log in first");return}this.uiController.showSendFleetModal(e)}handleTradeRouteAction(){const e=T.getSelectedSystem();if(!e){this.uiController.showError("Please select a system first");return}if(!$.isLoggedIn()){this.uiController.showError("Please log in first");return}this.uiController.showTradeRouteModal(e)}handleColonizeAction(){const e=T.getSelectedSystem();if(!e){this.uiController.showError("Please select a system first");return}if(!$.isLoggedIn()){this.uiController.showError("Please log in first");return}this.uiController.showColonizeModal(e)}getConnectedSystems(e){if(!e||!T.systems)return{};const t={},s=e.x,i=e.y;return T.systems.forEach(n=>{if(n.id===e.id)return;const o=n.x-s,a=n.y-i,r=Math.sqrt(o*o+a*a);if(r>800)return;const l=Math.atan2(a,o)*180/Math.PI;l>=-45&&l<=45?(!t.right||r<t.right.distance)&&(t.right={system:n,distance:r}):l>=45&&l<=135?(!t.down||r<t.down.distance)&&(t.down={system:n,distance:r}):l>=135||l<=-135?(!t.left||r<t.left.distance)&&(t.left={system:n,distance:r}):(!t.up||r<t.up.distance)&&(t.up={system:n,distance:r})}),t}navigateToSystem(e){var n,o;const t=T.getSelectedSystem();if(!t)return;const i=this.getConnectedSystems(t)[e];if(i&&i.system){T.selectSystem(i.system.id),this.mapRenderer.centerOnSystem(i.system.id);const a=((o=(n=T.mapData)==null?void 0:n.planets)==null?void 0:o.filter(r=>r.system_id===i.system.id))||[];this.uiController.displaySystemView(i.system,a)}}async sendFleetToSystem(e){var o;const t=T.getSelectedSystem();if(!t){this.uiController.showToast("Select a system first","error");return}if(!$.isLoggedIn()){this.uiController.showToast("Please log in to send fleets","error");return}const i=this.getConnectedSystems(t)[e];if(!i||!i.system){this.uiController.showToast(`No system found to the ${e}`,"error");return}if((((o=T.fleets)==null?void 0:o.filter(a=>{var r;return a.owner_id===((r=$.getUser())==null?void 0:r.id)&&a.current_system===t.id&&!a.destination_system}))||[]).length===0){this.uiController.showToast("No available fleets at this system","error");return}try{await w.sendFleet(t.id,i.system.id,10)&&(this.uiController.showToast(`🚀 Fleet dispatched to ${i.system.name||`System ${i.system.id.slice(-4)}`}`),this.mapRenderer.showFleetRoute(t,i.system))}catch(a){console.error("Failed to send fleet:",a),this.uiController.showToast(`Failed to send fleet: ${a.message||"Unknown error"}`,"error")}}async handleMultiMoveFleet(e,t){var o,a;if(!$.isLoggedIn()){this.uiController.showToast("Please log in to send fleets","error");return}if(e.owner_id!==((o=$.getUser())==null?void 0:o.id)){this.uiController.showToast("You don't own this fleet","error");return}if((a=T.fleetOrders)==null?void 0:a.find(r=>r.fleet_id===e.id&&(r.status==="pending"||r.status==="processing"))){this.uiController.showToast("Fleet already has pending orders","error");return}const i=this.mapRenderer.systems.find(r=>r.id===e.current_system);if(!i){this.uiController.showToast("Fleet's location not found","error");return}const n=this.findFleetPath(i,t);if(!n||n.length<2){this.uiController.showToast("No valid route found to target system","error");return}if(n.length===2)try{console.log(`🚀 Creating single-hop fleet order: ${i.name||i.id.slice(-4)} → ${t.name||t.id.slice(-4)}`);const r=await w.sendFleet(e.current_system,t.id,null,e.id);this.uiController.showToast(`Fleet order created: ${t.name||`System ${t.id.slice(-4)}`} (arrives in ~20s)`,"success"),console.log("Single-hop fleet order created:",r)}catch(r){console.error("Failed to create fleet order:",r),this.uiController.showToast(r.message||"Failed to create fleet order","error")}else try{console.log(`🚀 Creating multi-hop fleet route: ${n.length-1} hops`),console.log(`Route: ${n.map(c=>c.name||c.id.slice(-4)).join(" → ")}`);const r=n.map(c=>c.id),l=await w.sendFleetRoute(e.id,r);this.uiController.showToast(`Multi-hop route created: ${t.name||`System ${t.id.slice(-4)}`} (${n.length-1} hops, ~${(n.length-1)*20}s total)`,"success"),console.log("Multi-hop fleet route created:",l),this.fleetRoutes.set(e.id,{fullPath:n,currentHop:0,targetSystem:t,lastUpdate:Date.now(),isMultiHop:!0,totalHops:n.length-1}),this.mapRenderer.showFleetRoute(n,0)}catch(r){console.error("Failed to create multi-hop fleet route:",r),this.uiController.showToast(r.message||"Failed to create multi-hop fleet route","error")}}onFleetArrival(e){var s;console.log(`DEBUG: onFleetArrival called for fleet ${e}`);const t=this.fleetRoutes.get(e);if(console.log(`DEBUG: Route data for fleet ${e}:`,t),t&&t.isMultiHop){const i=(s=T.fleetOrders)==null?void 0:s.find(n=>n.fleet_id===e&&(n.status==="pending"||n.status==="processing"));if(i){const n=i.current_hop||0;t.currentHop=n,t.lastUpdate=Date.now(),console.log(`DEBUG: Updated route visualization for fleet ${e}, hop ${n}/${t.totalHops}`),this.mapRenderer.showFleetRoute(t.fullPath,n)}else console.log(`DEBUG: Fleet ${e} route completed, cleaning up visualization`),this.fleetRoutes.delete(e),this.mapRenderer.clearFleetRoute()}else console.log(`DEBUG: No multi-hop route data found for fleet ${e}`)}findFleetPath(e,t){var o;const s=new Set,i=[{system:e,path:[e]}],n=15;for(console.log(`🗺️ Pathfinding from ${e.name||e.id.slice(-4)} to ${t.name||t.id.slice(-4)}`);i.length>0;){const a=i.shift(),r=a.system;if(r.id===t.id)return console.log(`✅ Path found with ${a.path.length} hops:`,a.path.map(c=>c.name||c.id.slice(-4))),a.path;if(s.has(r.id)||a.path.length>n)continue;s.add(r.id);const l=((o=this.mapRenderer.systems)==null?void 0:o.filter(c=>c.id===r.id||s.has(c.id)?!1:this.mapRenderer.areSystemsConnected(r,c)))||[];console.log(`🔍 From ${r.name||r.id.slice(-4)}: found ${l.length} connected systems (hop ${a.path.length})`),l.forEach(c=>{s.has(c.id)||i.push({system:c,path:[...a.path,c]})})}return console.log(`❌ No path found from ${e.name||e.id.slice(-4)} to ${t.name||t.id.slice(-4)}`),null}calculatePathDistance(e){let t=0;for(let s=0;s<e.length-1;s++){const i=e[s],n=e[s+1],o=n.x-i.x,a=n.y-i.y,r=Math.sqrt(o*o+a*a);t+=r}return t}displaySelectedFleetInfo(e,t,s){this.uiController.displayFleetView(e,t,s)}handleKeyboardInput(e){if(!(e.target.tagName==="INPUT"||e.target.tagName==="TEXTAREA"))switch(e.key.toLowerCase()){case"escape":this.uiController.hideModal(),document.getElementById("context-menu").classList.add("hidden");break;case"arrowup":e.preventDefault(),e.shiftKey?this.sendFleetToSystem("up"):this.navigateToSystem("up");break;case"arrowdown":e.preventDefault(),e.shiftKey?this.sendFleetToSystem("down"):this.navigateToSystem("down");break;case"arrowleft":e.preventDefault(),e.shiftKey?this.sendFleetToSystem("left"):this.navigateToSystem("left");break;case"arrowright":e.preventDefault(),e.shiftKey?this.sendFleetToSystem("right"):this.navigateToSystem("right");break;case"f":this.handleSendFleetAction();break;case"t":this.handleTradeRouteAction();break;case"b":this.handleBuildAction();break;case"c":T.getSelectedSystem()&&this.mapRenderer.centerOnSystem(T.getSelectedSystem().id);break;case"o":this.handleColonizeAction();break;case"h":this.mapRenderer.fitToSystems();break}}}const it=new st;window.app=it;
