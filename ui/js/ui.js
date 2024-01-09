
function uniqueId(prefix){
    return `${prefix}${new Date().getTime()}`;
}

function actionsBuilder(){
    var actions=[];
    this.listActionEntry=(actnentryval,actnentrylbl)=>{
        if(actnentryval!==undefined&&actnentryval!==null) {
            if(actnentrylbl!==undefined&&actnentrylbl!==null&&typeof actnentrylbl==="string"){
                if((actnentrylbl=actnentrylbl.trim())!==""){
                    if(typeof actnentryval=="function") {
                        actions.push({"label":actnentrylbl,"event":actnentryval});
                    } else if (typeof actnentryval === "object" && !Array.isArray(actnentryval)){
                        if(actnentryval["label"]===undefined || actnentryval["label"]===null || actnentryval["label"]===""){
                            actnentryval["label"]=actnentrylbl;
                            actions.push(actnentryval);
                        }
                    }
                }
            } else if(actnentrylbl==undefined || actnentrylbl==null){
                if (typeof actnentryval === "object" && !Array.isArray(actnentryval)){
                    if(Object.keys(actnentryval).length===1){
                        Object.entries(actnentryval).forEach((actne)=>{
                            if(actne[1]!==undefined&&actne[1]!==null) {
                                if(typeof  actne[1]==="function") {
                                    actions.push({"label":actne[0],"event":actne[1]});
                                } else if(typeof actne[1]==="object"&&!Array.isArray(actne[1])){
                                    actne[1]["label"]=actne[0];
                                    actions.push(actne[1]);
                                }
                            }
                        });
                    } else {
                        actions.push(actnentryval);
                    }
                }
            }
        }
    }
    var i=0;
    var argsfound=[];    
    while((i<arguments.length)||argsfound.length>0){
        if(i<arguments.length){
            for(;i<arguments.length;i++){
                if(arguments[i]!==undefined&&arguments[i]!==null){
                    if (typeof arguments[i]==="function") {
                        arguments[i]=arguments[i]();
                        i--;
                    } else if (typeof arguments[i]==="object"){
                        if(Array.isArray(arguments[i])){
                            arguments[i].forEach((argo)=>{
                                argsfound.push(argo);
                            });
                        } else {
                            var argo=arguments[i];
                            if (i===0 && arguments.length-1===i){
                               if(argo["label"]===undefined||argo["label"]===null){
                                    if (argo["event"]===undefined||argo["event"]===null) {
                                        Object.entries(argo).forEach((argoe)=>{
                                            if((argoe[0]=argoe[0].trim())!=="" && argoe[1]!==undefined&&argoe[1]!==undefined){
                                                if (typeof argoe[1]==="function"){
                                                    argsfound.push({"label":argoe[0],"event":argoe[1]});
                                                } else if (typeof argoe[1]==="object"&&!Array.isArray(argoe[1])){
                                                    argoe[1]["label"]=argoe[0];
                                                    argsfound.push(argoe[1]);
                                                }
                                            }
                                        })
                                    } else {
                                        continue;
                                    }
                               } else {
                                    argsfound.push(argo);
                               }
                            } else{
                                argsfound.push(argo);
                            }
                        }
                    } else if(typeof arguments[i]==="string" && (arguments[i]=arguments[i].trim())!==""){
                        argsfound.push(arguments[i]);
                    }
                }        
            }
        }
        while(argsfound.length>0){
            var argv=argsfound.shift();
            if(argv!==undefined&&argv!==null){
                if(typeof argv==="string" && (argv=argv.trim())!==""){
                    actions.push({"label":argv});
                } else if (typeof argv==="object" && !Array.isArray(argv)){
                    this.listActionEntry(argv);
                }
            }
        }
    }
    this.buildActions=(container,buildAction)=>{
        this.buildActionsCollection=(ownerElm)=>{
            if(ownerElm instanceof HTMLElement){
                actions.forEach((crntactn,i)=>{
                    buildAction(crntactn,i,ownerElm);
                });
            }
        }
        if(container!==undefined&&container!==null&&buildAction!==undefined&&buildAction!==null&&typeof buildAction=="function"){
            if(typeof container==="string") {
                if(actions.length>0){
                    document.querySelectorAll(container).forEach((ownerelm,ownerelmi)=>{
                        this.buildActionsCollection(ownerelm);
                    });
                }
            } else if (container instanceof HTMLElement) {
                this.buildActionsCollection(container);
            }
        }
    };
    this.triggerAction=(action,actionEvent,triggerFind)=>{
        if (actions.length>0 && actionEvent!=undefined&&actionEvent!==null&&typeof actionEvent==="function"){
            var actionstocheck=[];
            if(action!==undefined&&action!==null){
                if(typeof action==="object"){
                    if(Array.isArray(action)){
                        action.forEach((actntochk)=>{
                            if(actntochk!==undefined&&actntochk!==null){
                                if ((typeof actntochk==="object" && !Array.isArray(actntochk))||(typeof actntochk==="string" &&(actntochk=actntochk.trim())!=="")){
                                    actionstocheck.push(actntochk);
                                }
                            }
                        });
                    } else {
                        actionstocheck.push(action);
                    }
                } else if(typeof action==="string" && (action=action.trim())!==""){
                    actionstocheck.push(action);
                }
            }
            
            var actnchki=0;
            while(actionstocheck.length>0){
                var actnsmtchd=0;
                actions.forEach((actn,actni)=>{
                    if(actionstocheck.length===0) return;
                    var crntactnchkng=actionstocheck[actnchki];
                    if (triggerFind!==undefined&&triggerFind!==null&&typeof triggerFind==="function"&&triggerFind(crntactnchkng,actn,actni)){
                        actionEvent(actn,actni,crntactnchkng);
                        actnsmtchd++;
                        actionstocheck.splice(actnchki,1);
                    } else if(typeof crntactnchkng==="string"&&actn["label"]!==undefined&&actn["label"]!==null&&actn["label"]===crntactnchkng){
                        actionEvent(actn,actni,crntactnchkng);
                        actionstocheck.splice(actnchki,1);
                        actnsmtchd++;
                    } else if(actnchki<actionstocheck.length){
                        actnchki++;
                    }
                });
                if(actnsmtchd==0){
                    break;
                } else {
                    actnchki=0;
                }
            }
        }
    }
    return this;
}

function dialogBuilder() {
    this.buildDialog=(options)=>{
        this.dlgid=uniqueId("dlg");
        this.dlgelm= document.createElement("div")
        this.dlgelm.setAttribute("id",this.dlgid);
        this.dlgelm.setAttribute("style","z-index:3;display:block;padding-top:100px;position:fixed;left:0;top:0;width:100%;height:100%;overflow:auto;background-color:rgb(0,0,0);background-color:rgba(0,0,0,0.4)");
        this.dlgelm.append(document.createElement("div"));
        this.dlgelm.children[0].setAttribute("style",`margin:auto;background-color:#fff;position:relative;padding:10px;outline:0;width:600px`);
        var dlgcntntelm=this.dlgelm.children[0];
        var dlgtitleelm=document.createElement("div");
        dlgcntntelm.append(dlgtitleelm);
        var dlgcontentelm=document.createElement("div");
        dlgcntntelm.append(dlgcontentelm);
        var dlgbuttonselm=document.createElement("div");
        dlgcntntelm.append(dlgbuttonselm);
        var btnsargs=[];
        var btnoptns={};
        var defaultbtnevent=null;
        if(options!==undefined&&options!==null&&typeof options==="object"&&!Array.isArray(options)){
            Object.entries(options).forEach((opte)=>{
                if ((opte[0]=opte[0].trim())!==""&&opte[1]!==undefined&&opte[1]!==null){
                    var opteval=opte[1];
                    if(typeof opteval==="function"){
                        opteval=opteval();
                    }
                    if (typeof opteval==="string"){
                        if(opte[0]==="title"){
                            dlgtitleelm.innerHTML=opteval;
                        } else if (opte[0]==="content") {
                            dlgcontentelm.innerHTML=opteval
                        } else if(opte[0]==="title-css"){
                            dlgtitleelm.setAttribute("style",opteval);
                        } else if (opte[0]==="content-css") {
                            dlgcontentelm.setAttribute("style",opteval);
                        } else if(opte[0]==="title-cssclass"){
                            dlgtitleelm.setAttribute("class",opteval);
                        } else if (opte[0]==="content-cssclass") {
                            dlgcontentelm.setAttribute("class",opteval);
                        } else if(opte[0]==="buttons-css"){
                            dlgbuttonselm.setAttribute("style",opteval);
                        } else if(opte[0]==="buttons-cssclass"){
                            dlgbuttonselm.setAttribute("class",opteval);
                        }
                    } else if (opte[0]==="buttons"){
                        if (typeof opteval==="string"){
                            btnsargs.push(opteval);
                        } else if (typeof opteval==="object"){
                            if(Array.isArray(opteval)) {
                                btnsargs.push(...opteval)
                            } else {
                                btnsargs.push(opteval);
                            }
                        }
                    } else if(opte[0]==="default-event"&&typeof opte[1]==="function"){
                        defaultbtnevent=opte[1];
                    }
                }
            });
        }
        this.removeDlg=function(){
            if(this.dlgelm!==undefined&&this.dlgelm!==null&&this.dlgelm instanceof Element){
                if(this.dlgelm.remove()){
                    this.dlgelm=null;
                }
            }  
        };
        btnoptns["wrapup-event"]=this.removeDlg;
        if(defaultbtnevent!==null){
            btnoptns["default-event"]=defaultbtnevent;
        }
        btnoptns["formref"]=dlgcntntelm;
        var buildButtons=buttonsBuilder(...btnsargs);
        buildButtons.buildButtons(dlgbuttonselm,btnoptns);
        document.body.appendChild(this.dlgelm);
    }
    return this;
}

function buttonsBuilder() {
    if(arguments.length>0){
        var buildActions=actionsBuilder(...arguments);
        this.buildButtonsCollections=(btnsOwner,options)=>{
            var defaultEvent=null;
            var defaultTarget=null;
            var defaultFormRefs=null;
            var defaultUrls=null;
            var defaultCss="";
            var defaultCssClass="";
            var defaultCssHover="";
            var defaultCssClassHover="";
            var defaultCssFocus="";
            var defaultCssClassFocus="";
            var focusEvent=null;
            var wrapupEvent=null;
            if(options!==undefined&&options!==null&&typeof options==="object"&&!Array.isArray(options)){
                Object.entries(options).forEach((opte)=>{
                    if(opte[1]!==undefined&&opte[1]!==null){
                        if(typeof opte[1] ==="function"){
                            if(opte[0]=="default-event"){
                                defaultEvent=opte[1];
                            } else if(opte[0]=="wrapup-event"){
                                wrapupEvent=opte[1];
                            } else if(opte[0]=="focus-event"){
                                focusEvent=opte[1];
                            }
                        } else if(typeof opte[1]==="string"&&(opte[1]=opte[1].trim())!=="") {
                            if(opte[0]=="css"){
                                defaultCss=opte[1];
                            } else if(opte[0]=="cssclass"){
                                defaultCssClass=opte[1];
                            } else if(opte[0]=="css-hover"){
                                defaultCssHover=opte[1];
                            } else if(opte[0]=="cssclass-hover"){
                                defaultCssClassHover=opte[1];
                            } else if(opte[0]=="css-focus"){
                                defaultCssFocus=opte[1];
                            } else if(opte[0]=="cssclass-focus"){
                                defaultCssClassFocus=opte[1];
                            } else {
                                if (defaultFormRefs===null&&opte[0]==="formref"){
                                    defaultFormRefs=opte[1];
                                } else if (defaultUrls===null&&opte[0]==="urlref"){
                                    defaultUrls=opte[1];
                                } else if (defaultTarget===null&&opte[0]==="target"){
                                    defaultTarget=opte[1];
                                }
                            }
                        } else if (opte[1] instanceof HTMLElement) {
                            if (defaultFormRefs===null&&opte[0]==="formref"){
                                defaultFormRefs=opte[1];
                            }
                        }
                    }
                });
            }
            if (btnsOwner instanceof HTMLElement) {
                while(btnsOwner.firstChild){
                    btnsOwner.removeChild(btnsOwner.firstChild);
                }
            }
            
            var btnstoAdd=[];

            var ownerid=btnsOwner.getAttribute("id");
            if(ownerid===null){
                ownerid=uniqueId("owner");
            }
            var ownerclassref=`#${ownerid} button`;
            var ownerbuttonstyling="";
            if(defaultCss!==""){
                ownerbuttonstyling+=`${ownerclassref} {${defaultCss}} `
            }
            if(defaultCssHover!==""){
                ownerbuttonstyling+=`${ownerclassref+":hover"} {${defaultCssHover}} `
            }
            if(defaultCssFocus!==""){
                ownerbuttonstyling+=`${ownerclassref+":focus"} {${defaultCssFocus}} `
            }
            buildActions.buildActions(btnsOwner,(actn,actni,elmOwner)=>{
                var btnelm=document.createElement("button");
                btnelm.setAttribute("type","button");
                var btnclass=defaultCssClass;
                var btnclassref=`btn${actni}`;
                var btncss=actn["css"]!==undefined&&actn["css"]!==null&&typeof actn["css"] === "string"&&(actn["css"]=actn["css"].trim())!==""?actn["css"]:"";
                if (btncss!==""){
                    ownerbuttonstyling+=`#${ownerid} .${btnclassref} {${btncss}}`;
                    if(!btnclass.includes(btnclassref)) btnclass+=btnclassref+" ";
                }
                var btncssclass=actn["cssclass"]!==undefined&&actn["cssclass"]!==null&&typeof actn["cssclass"] === "string"&&(actn["cssclass"]=actn["cssclass"].trim())!==""?actn["cssclass"]:"";
                if (btncssclass!==""){
                    btnclass+=btncssclass+" ";
                }
                var btncsshover=actn["css-hover"]!==undefined&&actn["css-hover"]!==null&&typeof actn["css-hover"] === "string"&&(actn["css-hover"]=actn["css-hover"].trim())!==""?actn["css-hover"]:"";
                if (btncsshover!==""){
                    ownerbuttonstyling+=`#${ownerid} .${btnclassref}:hover {${btncsshover}}`;
                    if(!btnclass.includes(btnclassref)) btnclass+=btnclassref+" ";
                }
                var btncssclasshover=actn["cssclass-hover"]!==undefined&&actn["cssclass-hover"]!==null&&typeof actn["cssclass-hover"] === "string"&&(actn["cssclass-hover"]=actn["cssclass-hover"].trim())!==""?actn["cssclass-hover"]:"";
                if (btncssclasshover!==""){
                    btnclass+=btncssclasshover+" ";
                }
                var btncssfocus=actn["css-focus"]!==undefined&&actn["css-focus"]!==null&&typeof actn["css-focus"] === "string"&&(actn["css-focus"]=actn["css-focus"].trim())!==""?actn["css-focus"]:"";
                if (btncsshover!==""){
                    ownerbuttonstyling+=`#${ownerid} .${btnclassref}:focus {${btncssfocuS}}`;
                    if(!btnclass.includes(btnclassref)) btnclass+=btnclassref+" ";
                }
                var btncssclassfocus=actn["cssclass-focus"]!==undefined&&actn["cssclass-focus"]!==null&&typeof actn["cssclass-focus"] === "string"&&(actn["cssclass-focus"]=actn["cssclass-focus"].trim())!==""?actn["cssclass-focus"]:"";
                if (btncssclassfocus!==""){
                    btnclass+=btncssclassfocus+" ";
                }
                if(btnclass!==""){
                    btnelm.setAttribute("class",btnclass.trim());
                }                
                var btntarget=actn["target"]!==undefined&&actn["target"]!==null?actn["target"]:defaultTarget;
                var btnurlref=actn["urlref"]!==undefined&&actn["urlref"]!==null?actn["urlref"]:defaultUrls;
                var btnformref=actn["formref"]!==undefined&&actn["formref"]!==null?actn["formref"]:defaultFormRefs;

                if(actn["event"]!==undefined&&actn["event"]!==null&&typeof actn["event"]==="function"){
                    btnelm.onclick=(ev)=>{
                        if(btnurlref!==null&&(actn["urlref"]===undefined||actn["urlref"]===null)){
                            actn["urlref"]=btnurlref;
                        }
                        if(btnformref!==null&&(actn["formref"]===undefined||actn["formref"]===null)){
                            actn["formref"]=btnformref;
                        }
                        if(btnurlref!==null&&btntarget!==null&&(actn["target"]===undefined||actn["target"]===null)){
                            actn["target"]=btntarget;
                        }
                        actn["event"](ev, actn,elmOwner);
                        if(wrapupEvent!=undefined&&wrapupEvent!==null&&typeof wrapupEvent){
                            wrapupEvent(ev, actn,elmOwner)
                        }
                    }
                } else if(defaultEvent!==undefined&&defaultEvent!==null&&typeof defaultEvent==="function") {
                    btnelm.onclick=(ev)=>{
                        if(btnurlref!==null&&(actn["urlref"]===undefined||actn["urlref"]===null)){
                            actn["urlref"]=btnurlref;
                        }
                        if(btnformref!==null&&(actn["formref"]===undefined||actn["formref"]===null)){
                            actn["formref"]=btnformref;
                        }
                        if(btnurlref!==null&&btntarget!==null&&(actn["target"]===undefined||actn["target"]===null)){
                            actn["target"]=btntarget;
                        }
                        defaultEvent(ev, actn,elmOwner);
                        if(wrapupEvent!=undefined&&wrapupEvent!==null&&typeof wrapupEvent){
                            wrapupEvent(ev, actn,elmOwner);
                        }
                    }
                } else {
                    btnelm.onclick=(ev)=>{
                        if(btnurlref!==null&&btnformref!==null){
                            if(btnurlref!==null&&(actn["urlref"]===undefined||actn["urlref"]===null)){
                                actn["urlref"]=btnurlref;
                            }
                            if(btnformref!==null&&(actn["formref"]===undefined||actn["formref"]===null)){
                                actn["formref"]=btnformref;
                            }
                            if(btnurlref!==null&&btntarget!==null&&(actn["target"]===undefined||actn["target"]===null)){
                                actn["target"]=btntarget;
                            }
                            parseEval(actn);
                        }
                        if(wrapupEvent!=undefined&&wrapupEvent!==null&&typeof wrapupEvent){
                            wrapupEvent(ev, actn,elmOwner);
                        }
                    }
                }
                if(actn["focus-event"]!==undefined&&actn["focus-event"]!==null&&typeof actn["focus-event"]==="function"){
                    btnelm.onfocus=(ev)=>{
                        actn["focus-event"](ev, actn,elmOwner);
                    }
                } else if(focusEvent!==undefined&&focusEvent!==null&&typeof focusEvent==="function") {
                    btnelm.onfocus=(ev)=>{
                        focusEvent(ev, actn,elmOwner);
                    }
                }
                var btnicon=actn["icon"]!==undefined&&actn["icon"]!==null&&typeof actn["icon"]==="string"?actn["icon"].trim():"";
                if (btnicon.startsWith("mdi ")||btnicon.startsWith("mdi-")){
                    if(btnicon.startsWith("mdi-")) {
                        btnicon="mdi "+btnicon;
                    }
                    btnicon=`<span class="${btnicon}"></span>`;
                }
                if(btnicon.startsWith("fa-")&&btnicon.split(" ").length>1){
                    btnicon=`<i class="${btnicon}"></i>`;
                }
                var btniconalign=actn["icon-align"]!==undefined&&actn["icon-align"]!==null&&typeof actn["icon-align"]==="string"?actn["icon-align"].trim():"left";
                btnelm.innerHTML=(btniconalign==="left"?(btnicon+" "):"")+((actn["label"]!==undefined&&actn["label"]!==null&&typeof actn["label"]==="string")?actn["label"]:"")+(btniconalign==="right"?(" "+btnicon):"");
                btnstoAdd.push(btnelm);                    
            });
            if(ownerbuttonstyling!==""){
                if(btnsOwner instanceof HTMLElement){
                    
                    btnsOwner.innerHTML=`<style>${ownerbuttonstyling}</style>`;
                }
            }
            btnstoAdd.forEach((btn)=>{
                btnsOwner.appendChild(btn);
            });
        };
        this.buildButtons=(container,options)=>{
            if(container!==undefined&&container!==null){
                if(typeof container==="string" && (container=container.trim())!==""){
                    document.querySelectorAll(container).forEach((btnsOwner)=>{
                        this.buildButtonsCollections(btnsOwner,options);
                    });
                } else if (container instanceof HTMLElement){
                    this.buildButtonsCollections(container,options);
                }
            }            
        };
    }
    return this;
}

function accordiansBuilder() {
    var accordianActions=actionsBuilder(...arguments);
    this.buildAccordiansCollection=(container,options)=>{
        if(container instanceof HTMLElement) {
            while(container.hasChildNodes()){
                container.firstChild.remove();
            }
            var accords=[];
            var accordcntnts=[];
            var accordstyles=[];
            var accrdcss="";
            var accrdcssclass="";
            var accrdctntcss="";
            var accrdctntcssclass="";
            if(options!==undefined&&options!==null&&typeof options==="object"&&!Array.isArray(options)){
                Object.entries(options).forEach((opte)=>{
                    if(opte[1]!==undefined&&opte[1]!==null){
                        if(typeof opte[1]==="string"&&(opte[1]=opte[1].trim())!==""){
                            if(opte[0]==="css"){
                                accrdcss=opte[1];
                            } else if(opte[0]==="cssclass"){
                                accrdcssclass=opte[1];
                            } else if(opte[0]==="content-css"){
                                accrdctntcss=opte[1];
                            } else if(opte[0]==="content-cssclass"){
                                accrdctntcssclass=opte[1];
                            }
                        }
                    }
                })
            }
            accordianActions.buildActions(container,(actn,actni,ownerelm)=>{
                var nxtAccrdbtn=document.createElement("button");
                nxtAccrdbtn.setAttribute("type","button");
                var accordstyle={"display":"block","width":"100%"};
                if(actn["cssclass"]!==undefined&&actn["cssclass"]!==null&&actn["cssclass"]){
                    if (typeof actn["cssclass"]==="object"){
                        if (Array.isArray(actn["cssclass"])){
                            nxtAccrdbtn.setAttribute("class",(accrdcssclass===""?"":(accrdcssclass+" "))+genMarkupSetting("class",...Array.isArray(actn["cssclass"])));
                        } else if(accrdcssclass!==null&&accrdcssclass!=="") {
                            nxtAccrdbtn.setAttribute("class",accrdcssclass);
                        }
                    } else if(accrdcssclass!==null&&accrdcssclass!=="") {
                        nxtAccrdbtn.setAttribute("class",accrdcssclass);
                    }
                } else if(accrdcssclass!==null&&accrdcssclass!=="") {
                    nxtAccrdbtn.setAttribute("class",accrdcssclass);
                }
                if(actn["css"]!==undefined&&actn["css"]!==null&&actn["css"]){
                    if (typeof actn["css"]==="object"){
                        if (Array.isArray(actn["css"])){
                            nxtAccrdbtn.setAttribute("style",(accrdcss===""?"":(accrdcss+";"))+genMarkupSetting("style",accordstyle)+";"+genMarkupSetting("style",...Array.isArray(actn["css"])));
                        } else {
                            Object.entries(actn["css"]).forEach((acccsse)=>{
                                if(acccsse[0]!=="width"&&acccsse[0]!=="display"){
                                    accordstyle[acccsse[0]]=acccsse[1];
                                }
                            });
                            nxtAccrdbtn.setAttribute("style",(accrdcss===""?"":(accrdcss+";"))+genMarkupSetting("style",accordstyle));
                        }
                    } else {
                        nxtAccrdbtn.setAttribute("style",(accrdcss===""?"":(accrdcss+";"))+genMarkupSetting("style",accordstyle));
                    }
                } else {
                    nxtAccrdbtn.setAttribute("style",(accrdcss===""?"":(accrdcss+";"))+genMarkupSetting("style",accordstyle));
                }
                var btnicon=actn["icon"]!==undefined&&actn["icon"]!==null&&typeof actn["icon"]==="string"?actn["icon"].trim():"";
                if (btnicon.startsWith("mdi ")||btnicon.startsWith("mdi-")){
                    if(btnicon.startsWith("mdi-")) {
                        btnicon="mdi "+btnicon;
                    }
                    btnicon=`<span class="${btnicon}"></span>`;
                }
                if(btnicon.startsWith("fa-")&&btnicon.split(" ").length>1){
                    btnicon=`<i class="${btnicon}"></i>`;
                }
                var btniconalign=actn["icon-align"]!==undefined&&actn["icon-align"]!==null&&typeof actn["icon-align"]==="string"?actn["icon-align"].trim():"left";
                nxtAccrdbtn.innerHTML=(btniconalign==="left"?(btnicon+" "):"")+((actn["label"]!==undefined&&actn["label"]!==null&&typeof actn["label"]==="string")?actn["label"]:"")+(btniconalign==="right"?(" "+btnicon):"");
                
                var nxtAccrdcntnt=document.createElement("div");
                var accordctntstyle={"display":"none","width":"100%"};
                if(actn["content-cssclass"]!==undefined&&actn["content-cssclass"]!==null&&actn["conntent-cssclass"]){
                    if (typeof actn["content-cssclass"]==="object"){
                        if (Array.isArray(actn["content-cssclass"])){
                            nxtAccrdcntnt.setAttribute("class",accrdctntcssclass===""?"":(accrdctntcssclass+" ")+genMarkupSetting("class",...Array.isArray(actn["content-cssclass"])));
                        } else if(accrdctntcssclass!==null&&accrdctntcssclass!=="") {
                            nxtAccrdcntnt.setAttribute("class",accrdctntcssclass);
                        }
                    } else if(accrdctntcssclass!==null&&accrdctntcssclass!=="") {
                        nxtAccrdcntnt.setAttribute("class",accrdctntcssclass);
                    }
                } else if(accrdctntcssclass!==null&&accrdctntcssclass!=="") {
                    nxtAccrdcntnt.setAttribute("class",accrdctntcssclass);
                }
                if(actn["content-css"]!==undefined&&actn["content-css"]!==null&&actn["content-css"]){
                    if (typeof actn["content-css"]==="object"){
                        if (Array.isArray(actn["content-css"])){
                            nxtAccrdcntnt.setAttribute("style",(accrdctntcss===""?"":(accrdctntcss+";"))+genMarkupSetting("style",accordctntstyle)+";"+genMarkupSetting("style",...Array.isArray(actn["content-css"])));
                        } else {
                            Object.entries(actn["content-css"]).forEach((acccsse)=>{
                                if(acccsse[0]!=="width"&&acccsse[0]!=="display"){
                                    accordctntstyle[acccsse[0]]=acccsse[1];
                                }
                            });
                            nxtAccrdcntnt.setAttribute("style",(accrdctntcss===""?"":(accrdctntcss+";"))+genMarkupSetting("style",accordctntstyle));
                        }
                    } else {
                        nxtAccrdcntnt.setAttribute("style",(accrdctntcss===""?"":(accrdctntcss+";"))+genMarkupSetting("style",accordctntstyle));
                    }
                } else {
                    nxtAccrdcntnt.setAttribute("style",(accrdctntcss===""?"":(accrdctntcss+";"))+genMarkupSetting("style",accordctntstyle));
                }      
                var accrdcntnt=actn["content"]!==undefined&&actn["content"]!==null?actn["content"]:null;
                if(accrdcntnt!==null){
                    if (typeof accrdcntnt==="string"&&(accrdcntnt=accrdcntnt.trim())!==""){
                        var arcrdcntnndes=document.querySelectorAll(accrdcntnt);
                        if(arcrdcntnndes.length==0) {
                            nxtAccrdcntnt.innerHTML=accrdcntnt;
                        } else {
                            arcrdcntnndes.forEach((accrdctntelm)=>{
                                nxtAccrdcntnt.append(accrdctntelm);
                            });
                        }
                    } else if (typeof accrdcntnt==="function"){
                        if((accrdcntnt=accrdcntnt(actn,nxtAccrdcntnt))!==undefined&&accrdcntnt!==null&&typeof accrdcntnt==="string"&&(accrdcntnt=accrdcntnt.trim())!==""){
                            nxtAccrdcntnt.innerHTML=nxtAccrdcntnt.innerHTML+accrdcntnt;
                        }
                    } else if (accrdcntnt instanceof HTMLElement) {
                        nxtAccrdcntnt.append(accrdcntnt);
                    }
                } else {
                    if(actn["urlref"]!==undefined&&actn["urlref"]!==null){
                        if(actn["target"]===undefined||actn["target"]===null){
                            actn["target"]=nxtAccrdcntnt;
                        }
                        alert(JSON.stringify(actn));
                        parseEval(actn);
                    }
                }
                nxtAccrdbtn.onclick=(e)=>{
                    accordcntnts.forEach((accordctnt)=>{
                        if(accordctnt instanceof HTMLElement){
                            if(nxtAccrdcntnt!==accordctnt) {
                                accordctnt.style.display="none";
                            } else {
                                nxtAccrdcntnt.style.display=nxtAccrdcntnt.style.display==="block"?"none":"block";
                            }
                        }
                    });
                };
                accords.push(nxtAccrdbtn)
                accordcntnts.push(nxtAccrdcntnt);
            });
            accords.forEach((accrdbtn,accrdbtni)=>{
                var accrdctnt=accordcntnts[accrdbtni];
                container.append(accrdbtn,accrdctnt);
            });
        
        }
    }
    this.buildAccordians=(container,options)=>{
        if(container!==undefined&&container!==null){
            if(typeof container==="string") {
                document.querySelectorAll(container).forEach((ownerElem)=>{
                    this.buildAccordiansCollection(ownerElem,options);
                });
            } else if (container instanceof HTMLElement) {
                this.buildAccordiansCollection(container,options);
            }
        }
    }
    return this;
}

function genMarkupSetting(){
    this.mrpstngcontent="";
    this.mrpstngnme="";
    this.mrpstngval=[];
    if(arguments.length>1){
        var strngargs=[];
        for(var i=0;i<arguments.length;i++){
            var arg=arguments[i];
            if(arg!==undefined&&arg!==null){
                if(typeof arg==="string" && (arg=arg.trim())!==""){
                    if(i==0){
                        this.mrpstngnme=arg;
                    } else {
                        strngargs.push(arg);
                    }
                } else if (i>0&&this.mrpstngnme!==""){
                    if(typeof arg==="object"){
                        if (Array.isArray(arg)){
                            if(arg.length>0){
                                strngargs=strngargs.concat(...arg);
                            }
                        } else {
                            
                            Object.entries(arg).forEach((stngentry)=>{
                                if((stngentry[0]=stngentry[0].trim())!=="" && (stngentry[1]!==undefined&&stngentry[1]!==null)){
                                    if(this.mrpstngnme=="style"){
                                        strngargs.push(stngentry[0]+":"+stngentry[1]);
                                    } else {
                                        stngestrngargsntry.push(stngentry[0]+"="+stngentry[1]);
                                    }
                                }
                            });
                        }
                    }
                } else {
                    break;
                }
            }
        }
        if(strngargs.length>0){
            if(this.mrpstngnme==="style"){
                this.mrpstngcontent=strngargs.join(";");
            } else if(this.mrpstngnme==="class"){
                this.mrpstngcontent=strngargs.join(" ");
            } else {
                this.mrpstngcontent=strngargs.join(";");
            }   
        }
    }
    return this.mrpstngcontent===""?"":`${this.mrpstngcontent}`;
}