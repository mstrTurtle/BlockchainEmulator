(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[506],{2027:function(e,t,n){Promise.resolve().then(n.bind(n,2540))},2540:function(e,t,n){"use strict";n.r(t),n.d(t,{default:function(){return q}});var r=n(7437),o=n(5430),s=n(2752);function l(){return(0,r.jsx)("h2",{children:"欢迎来到界面，点击按钮以连接Supervisor"})}var i=n(7394),c=n(8603),a=n(305),d=n(9944),u=n(5171),h=n(5771),g=n(7600);class m{async retryTillConnect(){for(console.log("in retryTillConnect");;)try{console.log("creating websocket"),new d.x;let e=(0,g.j)({url:"ws://localhost:7697"});return console.log("now racing for open"),e}catch(e){console.error("Connection failed"),this.fsm.toFailed()}}async receiveMsgLoop(e){let t=e.asObservable();console.log("in receiveMsgLoop"),t.pipe((0,h.h)(e=>"heartbeat"==e.type));let n=t.pipe((0,h.h)(e=>"heartbeat"!=e.type));for(;;){try{console.log("before receive msg");var r=await (0,u.z)(n);console.log("after receive msg,before fsm.next"),this.fsm.next(r)}catch(e){console.log("ERR happened: ",e),this.fsm.toFailed();return}if("bye"==r.type)return}}async start(){console.log("in start");let e=new WebSocket("ws://localhost:7697/api/status");e.onopen=t=>{console.log("opened"),e.onmessage=e=>{this.fsm.next(JSON.parse(e.data))}},e.onerror=e=>{console.log("error happened"),console.log(e),this.fsm.toFailed()}}constructor(){this.fsm=new v}}var p={type:"no_use",content:null};class v{init(e){return(0,a.z)(()=>{s.i7.val=s.qb.Init}),this.connecting}connecting(e){if(console.log("there is connecting. the m is:"),console.log(e),console.log("the m.type is ".concat(e.type)),"hello"==e.type)return(0,a.z)(()=>{s.i7.val=s.qb.Connected}),this.connected;throw Error()}connected(e){if("started"==e.type)return(0,a.z)(()=>{s.i7.val=s.qb.Running}),this.computing;throw Error()}computing(e){if("computing"==e.type)return(0,a.z)(()=>{s.YD.val=e.content}),this.computing;if("completed"==e.type)return(0,a.z)(()=>{s.i7.val=s.qb.Completed,s.Hj.val=e.content}),this.completed;throw Error()}completed(e){if("bye"==e.type)return this.completed;throw Error()}failed(e){throw Error()}toFailed(){(0,a.z)(()=>{s.i7.val=s.qb.Disconnected}),this.state=this.failed}next(e){console.log("FSM got Msg: ".concat(e)),console.log(e),console.log("FSM state before:"),console.log(this.state);try{this.state=this.state(e)}catch(e){console.log("FSM run into error catched:"),console.log(e)}console.log("FSM state after:"),console.log(this.state)}constructor(){this.state=this.init,this.next(p)}}var f=n(5760),x=n(2265);let b=()=>{console.log("start the receiver"),new m().start().then(()=>{console.log("Receiver Exit.")})},j=(0,o.Pi)(e=>{let{status:t}=e;return(0,x.useEffect)(()=>{b()},[]),(0,r.jsx)("div",{children:(0,r.jsxs)("h1",{children:["Emulator运行中 状态为",(0,r.jsx)("code",{className:"",children:s.qb[t.val]})]})})}),y=(0,o.Pi)(f.y),w=(0,o.Pi)(e=>{let{status:t}=e;return(0,r.jsx)("div",{children:(()=>{switch(t.val){case s.qb.Init:return(0,r.jsxs)(r.Fragment,{children:["初始状态 正在连接到Sup服务器。",(0,r.jsx)(l,{})]});case s.qb.Connected:return(0,r.jsx)(r.Fragment,{children:"连接成功 Sup还没开始运行"});case s.qb.ConnectionFailed:return(0,r.jsx)(r.Fragment,{children:"连接失败。请关闭重启。"});case s.qb.Running:return(0,r.jsxs)("div",{className:"rounded-lg border-2 border-blue-500 shadow",children:[(0,r.jsx)(i.Z,{wrap:"wrap",gap:"middle",style:{marginTop:16},children:(0,r.jsx)(c.Z,{type:"dashboard",steps:8,percent:s.YD.val.count/s.YD.val.total*100,trailColor:"rgba(0, 0, 0, 0.06)",strokeWidth:20})}),(0,r.jsx)("div",{children:"正在运行中:"}),(0,r.jsxs)("div",{children:["有",s.YD.val.count,"个交易处理完毕"]}),(0,r.jsxs)("div",{children:["这次运行的总量为",s.YD.val.total,"个交易。"]})]});case s.qb.RunningFailed:return(0,r.jsx)(r.Fragment,{children:"运行失败，请关闭重启。"});case s.qb.Completed:return(0,r.jsxs)("div",{className:"rounded-lg border-2 border-blue-500 shadow",children:[(0,r.jsx)("div",{className:"border-8 rounded-lg ",children:"运行完毕"}),(0,r.jsxs)("article",{className:"text-wrap rounded-lg border-8 shadow bg-white",children:[(0,r.jsx)("h3",{className:"text-2xl border-white ",children:"报告"}),(0,r.jsx)("div",{className:"rounded-lg border-white border-8",children:(0,r.jsx)(y,{report:s.Hj})})]})]});case s.qb.Disconnected:return(0,r.jsx)(r.Fragment,{children:"已退出"});default:return(0,r.jsx)(r.Fragment,{})}})()})});var F=()=>(0,r.jsxs)("div",{className:"absolute left-1/2 top-1/2 translate-x-[-50%] translate-y-[-50%]",children:[(0,r.jsx)(j,{status:s.i7}),(0,r.jsx)(w,{status:s.i7})]}),C=n(7138),N=n(9887),S=n(7449),k=n.n(S);function q(){return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(k(),{children:(0,r.jsx)("title",{children:"Blockemulator Supervisor Monitor"})}),(0,r.jsx)(C.default,{href:"https://github.com",children:(0,r.jsx)(N.Z,{})}),(0,r.jsx)(F,{})]})}},5760:function(e,t,n){"use strict";n.d(t,{y:function(){return d}});var r=n(7437),o=n(2752),s=n(8924),l=n(9939);let i=()=>(console.log(o.Hj),(0,r.jsx)(r.Fragment,{})),c=[{title:"测度名",dataIndex:"name",key:"name"},{title:"测度值",dataIndex:"vals",key:"vals"}],a=e=>e.map((e,t)=>{let{name:n,vals:r}=e;return{name:n,vals:JSON.stringify(r),key:t}}),d=e=>{let{report:t}=e;return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(s.ZP,{type:"primary",onClick:()=>{var e,n;e="data:text/json;charset=utf-8,"+encodeURIComponent(JSON.stringify(t.val)),(n=document.createElement("a")).setAttribute("href",e),n.setAttribute("download","report.json"),document.body.appendChild(n),n.click(),n.remove()},children:"下载json"}),(0,r.jsx)("div",{className:"font-bold",children:"Shard结果："}),(0,r.jsx)("div",{className:"font-mono",children:JSON.stringify(t.val.pbftShardCsv)}),(0,r.jsx)("hr",{className:"h-8"}),(0,r.jsx)("div",{className:"font-bold",children:"测度输出"}),i(),(0,r.jsx)(l.Z,{dataSource:a(t.val.measureOutputs),columns:c}),(0,r.jsx)("div",{className:"",children:"总的值"}),(0,r.jsx)("div",{className:"font-mono",children:JSON.stringify(t.val.measureOutputs)})]})}},2752:function(e,t,n){"use strict";n.d(t,{Hj:function(){return c},YD:function(){return i},i7:function(){return l},qb:function(){return o}});var r,o,s=n(305);(r=o||(o={}))[r.Init=0]="Init",r[r.Connected=1]="Connected",r[r.ConnectionFailed=2]="ConnectionFailed",r[r.Running=3]="Running",r[r.RunningFailed=4]="RunningFailed",r[r.Completed=5]="Completed",r[r.Disconnected=6]="Disconnected";let l=(0,s.ky)({val:0}),i=(0,s.ky)({val:{count:0,total:0}}),c=(0,s.ky)({val:{pbftShardCsv:[],measureOutputs:[]}})}},function(e){e.O(0,[147,710,332,924,21,632,103,971,23,744],function(){return e(e.s=2027)}),_N_E=e.O()}]);