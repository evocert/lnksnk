
function decodeHTMLEntities(text) {
    var entities = [
        ['amp', '&'],
        ['apos', '\''],
        ['#x27', '\''],
        ['#x2F', '/'],
        ['#39', '\''],
        ['#47', '/'],
        ['lt', '<'],
        ['gt', '>'],
        ['nbsp', ' '],
        ['quot', '"']
    ];

    for (var i = 0, max = entities.length; i < max; ++i)
        text = text.replace(new RegExp('&' + entities[i][0] + ';', 'g'), entities[i][1]);

    return text;
}

function convertHTMLEntity(text){
    const span = document.createElement('span');

    return text
    .replace(/&[#A-Za-z0-9]+;/gi, (entity,position,text)=> {
        span.innerHTML = entity;
        return span.innerText;
    });
}

function getAllUrlParams(url) {
    // get query string from url (optional) or window
    var queryString = url ? url.split('?')[1] : "";
    // we'll store the parameters here
    var obj = {};
    // if query string exists
    if (queryString) {
        // stuff after # is not part of query string, so get rid of it
        queryString = queryString.split('#')[0];
        // split our query string into its component parts
        var arr = queryString.split('&');
        for (var i = 0; i < arr.length; i++) {
            // separate the keys and the values
            var a = arr[i].split('=');
            // set parameter name and value (use 'true' if empty)
            var paramName = decodeURIComponent(a[0]);
            var paramValue = typeof (a[1]) === undefined ? "" : decodeURIComponent(a[1]);

            // if the paramName ends with square brackets, e.g. colors[] or colors[2]
            if (paramName.match(/\[(\d+)?\]$/)) {
                // create key if it doesn't exist
                var key = paramName.replace(/\[(\d+)?\]/, '');
                if (!obj[key]) obj[key] = [];
                // if it's an indexed array e.g. colors[2]
                if (paramName.match(/\[\d+\]$/)) {
                    // get the index value and add the entry at the appropriate position
                    var index = /\[(\d+)\]/.exec(paramName)[1];
                    obj[key][index] = paramValue;
                } else {
                    // otherwise add the value to the end of the array
                    obj[key].push(paramValue);
                }
            } else {
                // we're dealing with a string
                if (!obj[paramName]) {
                    // if it doesn't exist, create property
                    obj[paramName] = paramValue;
                } else if (obj[paramName] && typeof obj[paramName] === 'string'){
                    // if property does exist and it's a string, convert it to an array
                    obj[paramName] = [obj[paramName]];
                    obj[paramName].push(paramValue);
                } else {
                    // otherwise add the property
                    obj[paramName].push(paramValue);
                }
            }
        }
    }
    return obj;
}

var invalidinputfields={"button":true,"reset":true,"submit":true,"image":true};

function buildFormData(){
    var crntfrmdata=null;
    var frmargs=[];
    Array.from(arguments).forEach((arg)=>{
        if(arg!==undefined&&arg!==null){
            if(typeof arg==="string"&&arg!==""){
                arg.trim().split(",").forEach((ar)=>{
                if((ar=ar.trim())!==""){
                    frmargs.push(ar);
                } 
                });
            } else if (arg instanceof HTMLElement){
                frmargs.push(arg);
            }
        }
    });
    frmargs.forEach((elm)=>{
        if(elm!==undefined&&elm!==null){
            if (typeof elm==="string" && elm!=="") {
                document.querySelectorAll(elm+" input").forEach((inputelm)=>{
                    var inname="";
                    if((inname=inputelm.getAttribute("name")!==null&&inputelm.getAttribute("name")!==""?inputelm.getAttribute("name"):"")!==""){
                        var intype=inputelm.getAttribute("type")===null?"text":inputelm.getAttribute("type");
                        if (invalidinputfields[intype]===undefined ||!invalidinputfields[intype]){
                            if(intype==="file"){
                                if(crntfrmdata===null){
                                    crntfrmdata=new FormData();
                                }
                                if(inputelm.files.length>0){
                                    for(var fi=0;fi<inputelm.files.length;fi++){
                                        crntfrmdata.append(inname,inputelm.files[fi]);
                                    } 
                                }                            
                            } else {
                                var invalue=inputelm.value;
                                if(invalue===null){
                                    invalue="";
                                }
                                if(crntfrmdata===null){
                                    crntfrmdata=new FormData();
                                }
                                crntfrmdata.append(inname,invalue);
                            }   
                        }
                    }
                });
                document.querySelectorAll(elm+" textarea").forEach((txtareaelm)=>{
                    var inname= txtareaelm.getAttribute("name");
                    if(inname!==null&&inname!==""){
                        var invalue=convertHTMLEntity(txtareaelm.innerHTML);
                        if(crntfrmdata===null){
                            crntfrmdata=new FormData();
                        }
                        crntfrmdata.append(inname,invalue);
                    }
                });
            } else if (elm instanceof HTMLElement){
                elm.querySelectorAll("input").forEach((inputelm)=>{
                    var inname="";
                    if((inname=inputelm.getAttribute("name")!==null&&inputelm.getAttribute("name")!==""?inputelm.getAttribute("name"):"")!==""){
                        var intype=inputelm.getAttribute("type")===null?"text":inputelm.getAttribute("type");
                        if (invalidinputfields[intype]===undefined ||!invalidinputfields[intype]){
                            if(intype==="file"){
                                if(crntfrmdata===null){
                                    crntfrmdata=new FormData();
                                }
                                if(inputelm.files.length>0){
                                    for(var fi=0;fi<inputelm.files.length;fi++){
                                        crntfrmdata.append(inname,inputelm.files[fi]);
                                    } 
                                }                            
                            } else {
                                var invalue=inputelm.value;
                                if(invalue===null){
                                    invalue="";
                                }
                                if(crntfrmdata===null){
                                    crntfrmdata=new FormData();
                                }
                                crntfrmdata.append(inname,invalue);
                            }   
                        }
                    }
                });
                elm.querySelectorAll("textarea").forEach((txtareaelm)=>{
                    var inname= txtareaelm.getAttribute("name");
                    if(inname!==null&&inname!==""){
                        var invalue=convertHTMLEntity(txtareaelm.innerHTML);
                        if(crntfrmdata===null){
                            crntfrmdata=new FormData();
                        }
                        crntfrmdata.append(inname,invalue);
                    }
                });
            } else if(typeof elm==="object"){
                var elmarr=[];
                if(Array.isArray(elm)){
                    elmarr.push(...elm);
                } else {
                    elmarr.push(elm);
                }
                elmarr.forEach((elme)=>{
                    if (elme!==undefined&&elme!==null&&typeof elme==="object"&&!Array.isArray(elme)){
                        Object.entries(elme).forEach((eelm)=>{
                            if(crntfrmdata===null){
                                crntfrmdata=new FormData();
                            }
                            crntfrmdata.append(eelm[0],eelm[1]);
                        });
                    }
                })
            }
        }
    });
    return crntfrmdata;
}

function post(){
    var options=null;
    if(arguments.length===1){
        if(arguments[0]!==undefined&&arguments[0]!==null){
            if(typeof arguments[0]==="string" && arguments[0]!==""){
                options={"url":arguments[0]+""};
            } else if(typeof arguments[0]==="function") {
                var options=arguments[0]();
                if(typeof options==="string") {
                    options={"url":options+""};
                }
            } else if(typeof arguments[0] ==="object") {
                options={};
                Object.entries(arguments[0]).forEach((k,v)=>{
                    options[v]=k;
                });
            }
        }
    }
}

function parse(options){
    var startParsing=options["start"]!==undefined&&options["start"]!==null&&typeof options["start"] === "function"?options["start"]:function(){};
    var doneParsing=options["end"]!==undefined&&options["end"]!==null&&typeof options["end"] === "function"?options["end"]:function(){};
    var print=options["print"]!==undefined&&options["print"]!==null&&typeof options["print"] === "function"?options["print"]:function(){};
    var write=options["write"]!==undefined&&options["write"]!==null&&typeof options["write"] === "function"?options["write"]:function(){};
    var evalactive=options["eval"]!==undefined&&options["eval"]!==null&&typeof options["eval"] === "function"?options["eval"]:function(){};
    var prepostlbl={"pre":options["prelbl"]!==undefined&&options["prelbl"]!==null&&typeof options["prelbl"] === "string"?options["prelbl"]:"<%",
                    "post":options["postlbl"]!==undefined&&options["postlbl"]!==null&&typeof options["postlbl"] === "string"?options["postlbl"]:"%>"};
    var unparsedcontents=options["template"]!==undefined&&options["template"]!==null&&typeof options["template"] === "string"?[options["template"]]:options["template"]!==undefined&&options["template"]!==null&&typeof options["template"] === "function"?options["template"]():options["template"]!==undefined&&options["template"]!==null?options["template"]:[];
    var activecode="";
    function doParser(){
        var mxprel=prepostlbl["pre"].length;
        var mxpostl=prepostlbl["post"].length;
        var isatvlvl=false;

        function capturePrint(capcontent){
            var endatv=activecode.trimEnd();
            if (endatv.length>0){
                if (["(","[","+","=",","].includes(endatv.substring(endatv.length-1))){
                    activecode+=`\`${capcontent}\``;
                } else {
                    activecode+=`print(\`${capcontent}\`);`;
                }
            } else {
                activecode+=`print(\`${capcontent}\`);`;
            }
        }
        for(var i=0;i<arguments.length;i++){
            var args=arguments[i]+"";
            var argsl=args.length;
            var lstpren=-1;
            var lsttargetn=-1;
            var lstpostn=-1;
            
            while(args.length>0) {
                if(isatvlvl){
                    if((lstpostn=args.indexOf(prepostlbl["post"]))>-1){
                        isatvlvl=false;
                        var postarg=args.substring(0,lstpostn);
                        if(postarg!==""){
                            if(postarg.includes("[#target:")){
                                while((lsttargetn=postarg.indexOf("[#target:"))>-1){
                                    if (postarg.substring(lsttargetn+"[#target:".length).indexOf("#]")>0){
                                        activecode+=postarg.substring(0,lsttargetn);
                                        postarg=postarg.substring(lsttargetn+"[#target:".length);
                                        activecode+=`_target(${postarg.substring(0,postarg.indexOf("#]"))})`;
                                        postarg=postarg.substring(postarg.indexOf("#]")+"#]".length);
                                    } else {
                                        break
                                    }
                                }
                                if (postarg!==""){
                                    activecode+=postarg;
                                }
                            } else {
                                activecode+=postarg;
                            }
                        }
                        args=args.substring(lstpostn+mxpostl);
                        continue;
                    }
                } else {
                    if((lstpren=args.indexOf(prepostlbl["pre"]))>-1){
                        isatvlvl=true;
                        var prearg=args.substring(0,lstpren);
                        if (prearg!==""){
                            if(activecode===""){
                                print(prearg);
                            } else {
                                //activecode+=`print(\`${prearg}\`);`;
                                capturePrint(prearg);
                            }
                        }
                        args=args.substring(lstpren+mxprel);
                    } else {
                        var psvarg=args;
                        if(activecode===""){
                            print(psvarg);
                        } else {
                            capturePrint(psvarg);
                        }
                        break;
                    }
                }
            }
        }
    }

    if (Array.isArray(unparsedcontents) && unparsedcontents.length>0) {
        activecode="";
        startParsing();
        unparsedcontents.forEach((unparsed)=>{
            doParser(unparsed)
        });
        if(activecode!==""){
            evalactive(activecode);
        }
        doneParsing();
    }
}

function elemAttributes(elm,elmattrs){
    if(typeof elmattrs ==="string") {
        elmattrs=elmattrs.trim();
        while(elmattrs!=""){
            var attrnme="";
            if(elmattrs.indexOf("=")>0){
                attrnme=elmattrs.substring(0,elmattrs.indexOf("=")).trim();
                if(attrnme.startsWith(`"`)&&elmattrs.endsWith(`"`)){
                    attrnme=attrnme.substring(1,attrnme.length-1);
                } else if(attrnme.startsWith(`'`)&&attrnme.endsWith(`'`)){
                    attrnme=attrnme.substring(1,attrnme.length-1);
                }
                elmattrs=elmattrs.substring(elmattrs.indexOf("=")+1).trim();
                if (attrnme!==""&&elmattrs!==""){
                    var txtpar=elmattrs.substring(0,1);
                    var attrval="";
                    if (txtpar===`"`||txtpar===`'`){
                        if((elmattrs=elmattrs.substring(1).trim()).indexOf(txtpar)>-1){
                            attrval=elmattrs.substring(0,elmattrs.indexOf(txtpar));
                            elmattrs=elmattrs.substring(elmattrs.indexOf(txtpar)+txtpar.length).trim();
                            if(attrval!=""){
                                elm.setAttribute(attrnme,attrval);
                            } else {
                                break;
                            }
                        } else {
                            break;
                        }
                    } else if((attrval=elmattrs.substring(0,elmattrs.indexOf(" ")>0?elmattrs.indexOf(" "):elmattrs.length).trim())!=="") {
                        elm.setAttribute(attrnme,attrval);
                        if (elmattrs.indexOf(" ")>0){
                            elmattrs=elmattrs.substring(elmattrs.indexOf(" ")+1).trim();
                        } else {
                            break;
                        }
                    } else {
                        break;
                    }
                } else {
                    break;
                }
            } else {
                break;
            }
        }
    }
}

const sleepSync = (ms) => {
    const end = new Date().getTime() + ms;
    while (new Date().getTime() < end) { /* do nothing */ }
  }

function parseEval(){
    if (arguments.length>0) {
       for(var i=0;i<arguments.length;i++){
            var arg=arguments[i];
            if(arg!==undefined&&arg!==null){
                if (arg instanceof HTMLElement){
                    _parseEval(arg);
                } else if(typeof arg ==="object" && !Array.isArray(arg)) {
                    _parseEval(arg);
                }
            }
       }
    } else {
        _parseEval();
    }
}

function prepTargetContent(targetelem, cntnttoprep){
    if (targetelem instanceof HTMLElement) {
        targetelem.innerHTML=cntnttoprep;
        targetelem.querySelectorAll("script").forEach((elm)=>{
            if (elm instanceof HTMLScriptElement) {
               var script=elm.innerHTML;
                var atti=0;
                var attl=elm.attributes.length;
                var scrptelm=document.createElement("script");
                while(atti<attl) {
                    var attr=elm.attributes.item(atti++);
                    scrptelm.setAttribute(attr.name,attr.value);
                }
                if(script!==undefined&&script!==null) {
                    scrptelm.innerHTML=script;   
                }
                elm.parentNode.replaceChild(scrptelm,elm);
            }
        })        
    } else if (targetelem===undefined||targetelem===null||targetelem==="") {
        
    }
    return targetelem;
}

function _parseEval(){
    var settings={};
    if (arguments.length==1&&arguments[0]!==undefined&&arguments[0]!==null) {
        if (arguments[0] instanceof HTMLElement){
            if(arguments[0].attributes.length>0){
                for(var attri=0;attri<arguments[0].attributes.length;attri++){
                    var attr=arguments[0].attributes[attri];
                    if(attr.name==="target"||attr.name=="formref"||attr.name==="urlref"||attr.name==="template"||attr.name==="jsonref") {
                        settings[attr.name]=attr.value;
                    }
                }
            }
        } else if(typeof arguments[0] === "object" && !Array.isArray(arguments[0])) {
            Object.entries(arguments[0]).forEach((entry)=>{
                if(entry[0]==="target"||entry[0]=="formref"||entry[0]==="urlref"||entry[0]==="template"||entry[0]==="jsonref") {
                    settings[entry[0]]=entry[1];
                }
            });
        }
    }
    var source=settings["template"]!==undefined&&settings["template"]!==null?settings["template"]:"";
    var israwscript=settings["template"]!==undefined&&settings["template"]!==null?true:false;
    var isdocscrpt=(settings["template"]!==undefined&&settings["template"]!==null)&&!israwscript&&document.currentScript!==undefined&&document.currentScript!==null;
    var sourceElm=isdocscrpt?document.currentScript:null;
    var sourceElmContent=sourceElm!==null?convertHTMLEntity(sourceElm.innerHTML).trim():israwscript?source:"";
    if(isdocscrpt&&(sourceElmContent!==""&&(sourceElmContent.endsWith("parseEval();")||sourceElmContent.endsWith("parseEval()"))&&!(sourceElmContent==="parseEval();"||sourceElmContent==="parseEval()"))){
        sourceElmContent=sourceElmContent.substring(0,sourceElmContent.length-(sourceElmContent.endsWith("parseEval();")?"parseEval();".length:sourceElmContent.endsWith("parseEval()")?"parseEval()".length:0)).trim();
    }
    var target=settings["target"]!==undefined&&settings["target"]!==null?settings["target"]:sourceElm!==null?sourceElm.getAttribute("target"):"";
    if(target===null){
        target="";
    }
    var jsonref=settings["jsonref"]!==undefined&&settings["jsonref"]!==null?settings["jsonref"]:sourceElm!==null?sourceElm.getAttribute("jsonref"):null;
    if (jsonref!==undefined&&jsonref!==null&&typeof jsonref==="object") {
        jsonref=JSON.parse(JSON.stringify(jsonref));
    }
    var formsrefs=settings["formref"]!==undefined&&settings["formref"]!==null?settings["formref"]:sourceElm!==null?sourceElm.getAttribute("formref"):"";
    if (formsrefs!==undefined&&formsrefs!==null){
        if (formsrefs instanceof HTMLElement) {
            formsrefs=[formsrefs];
        } else if(typeof formsrefs ==="string") {
            formsrefs=formsrefs.split(",");
        } else {
            formsrefs=[];
        }
    } else {
        formsrefs=[];
    }
    var urlref=settings["urlref"]!==undefined&&settings["urlref"]!==null?settings["urlref"]:sourceElm!==null?sourceElm.getAttribute("urlref"):null;
    if(urlref!==null){
        if(typeof urlref ==="string" && (urlref=urlref.trim())!=="") {
            urlref=[urlref];
        } else if(typeof urlref==="function" && ((urlref=urlref())!==undefined&&urlref!==null)){
            url=(typeof urlref ==="string"&&(urlref=urlref.trim())!=="")?[urlref]:(typeof urlref ==="object" && Array.isArray(urlref))?urlref:[];
        } else {
            urlref=[];
        }
    } else {
        urlref=[];
    }
    var template=[];
    if(sourceElm!==null){
        while(sourceElmContent!==""){       
            if(sourceElmContent.indexOf(isdocscrpt?"/*":"<!--")>-1){
                if((sourceElmContent=sourceElmContent.substring(sourceElmContent.indexOf(isdocscrpt?"/*":"<!--")+(isdocscrpt?"/*":"<!--").length)).indexOf(isdocscrpt?"*/":"-->")>-1){
                    template.push(sourceElmContent.substring(0,sourceElmContent.indexOf(isdocscrpt?"*/":"-->")));
                    sourceElmContent=sourceElmContent.substring(sourceElmContent.indexOf(isdocscrpt?"*/":"-->")+(isdocscrpt?"*/":"-->").length)
                } else {
                    break;
                }
            } else {
                break;
            }
        }
    } else if (israwscript) {
        template.push(sourceElmContent);
    } else if(urlref.length>0) {
        var frmdata=buildFormData(...formsrefs);
        urlref.forEach((urlrf)=>{
            var doclocation=document.location.href;
            var doclocationroot=doclocation;
            if(doclocation.lastIndexOf("://")<doclocation.lastIndexOf("/")-"://".length){
                doclocationroot=doclocation.substring(doclocation.lastIndexOf("://")+"://".length);
                doclocationroot=doclocation.substring(0,doclocation.lastIndexOf("://")+"://".length)+doclocationroot.substring(0,doclocationroot.indexOf("/"));
                if(!doclocation.endsWith("/")){
                    doclocation+="/"
                } else {
                    doclocation=doclocation.substring(0,doclocation.lastIndexOf("/")+1);
                }
            }
            if(urlrf.indexOf("/")==-1){
                urlrf=doclocation+urlrf;
            } else {
                urlrf=doclocationroot+urlrf;
            }
            var xhttp = new XMLHttpRequest();
            xhttp.onreadystatechange = function() {
                if (this.readyState == 4 && this.status == 200) {
                    template.push(this.responseText);
                }
            };
            xhttp.onerror==function(){
                
            };
            if (jsonref!==null) {
                if (jsonref instanceof HTMLInputElement || jsonref instanceof HTMLTextAreaElement) {
                    jsonref=jsonref.value;
                } else if (typeof jsonref==="object" || Array.isArray(jsonref)) {
                    jsonref=JSON.stringify(jsonref);
                } else if (typeof jsonref==="function") {
                    jsonref=jsonref();
                } else if (typeof jsonref!=="string"){
                    jsonref=null;
                }
            }
            if(frmdata!==null||jsonref!==null) {
                xhttp.open("POST",urlrf,false);
                if (jsonref!==null){
                    xhttp.setRequestHeader("Content-Type", "application/json; charset=UTF-8");
                    xhttp.send(jsonref);
                } else if (frmdata!==null){
                    xhttp.send(frmdata);
                }
            } else {
                xhttp.open("GET",urlrf,false);
                xhttp.send();
            }
        });
    }

    if(template.length>0){
        processContent(template);
    }

    function processContent(template) {
        var conttentprepped="";
        var print=function(){
            for(var i=0;i<arguments.length;i++) conttentprepped+=arguments[i]+"";
        };

        var reset=function(){conttentprepped="";}
        
        var write=function(ctnttowrite){
            if(ctnttowrite!==undefined&&ctnttowrite!==null&&(ctnttowrite=ctnttowrite.trim())!=="") {
                if(target===null||target===undefined||target==="") {
                    if(ctnttowrite.startsWith("<")&&ctnttowrite.endsWith(">")&&!ctnttowrite.endsWith("/>")&&ctnttowrite.indexOf(">")>0&&ctnttowrite.indexOf(">")<ctnttowrite.indexOf("</")){
                        var possibletag=ctnttowrite.substring("<".length,ctnttowrite.indexOf(">"));
                        var elmname="";
                        var elmrmng="";
                        var pn=0;
                        for(let pc in [...possibletag]){
                            if(possibletag.substring(pn,pn+1).trim()===""){
                                pn=pc;
                                break
                            }
                            pn++;
                        }
                    
                        elmname=possibletag.substring(0,pn).trim();
                        elmrmng=possibletag.substring(pn).trim();
                        if(ctnttowrite.startsWith("<"+elmname)&&ctnttowrite.endsWith("</"+elmname+">")){
                            ctnttowrite=ctnttowrite.substring(ctnttowrite.indexOf(">")+">".length,ctnttowrite.length-("</"+elmname+">").length);
                            if(isdocscrpt&&sourceElm!==undefined&&sourceElm!==null){
                                var newElm=document.createElement(elmname);
                                if(elmrmng!==""){
                                    elemAttributes(newElm,elmrmng);
                                }
                                prepTargetContent(newElm,ctnttowrite);
                                sourceElm.parentNode.replaceChild(newElm,sourceElm);
                            }
                        }
                    }
                } else if (target!==undefined&&target!==null){
                    if (typeof target==="string" && (target=target.trim())!==""){
                        if (target.startsWith("scrpt:")){
                            target=target.substring("scrpt:".length);
                            if (target==="") {
                                eval(ctnttowrite);
                            } else if (target.includes("[#code#]")) {
                                eval(target.substring(0,target.indexOf("[#code#]"))+ctnttowrite+target.substring(target.indexOf("[#code#]")+"[#code#]".length));
                            }
                        } else {
                            document.querySelectorAll(target).forEach((trgtelm)=>{
                                if(isdocscrpt&&sourceElm!==nul){
                                    for(var ai=0;ai<sourceElm.attributes.length;ai++){
                                        var attnme=sourceElm.attributes[ai].name;
                                        if(attnme==="target"||attnme==="urlref") continue;
                                        trgtelm.setAttribute(sourceElm.attributes[ai].name,sourceElm.attributes[ai].value);
                                    }
                                }
                                prepTargetContent(trgtelm,ctnttowrite);
                            });
                        }
                    } else if (target instanceof HTMLElement){
                        prepTargetContent(target,ctnttowrite);
                    }
                    if(isdocscrpt){
                        if(sourceElm!==undefined&&sourceElm!==null){
                            sourceElm.remove();
                        }
                    }
                }
            }
        };

        var _flush=function(){
            write();
            reset();
        }

        var _target=function(){
            if(arguments.length==1&&arguments[0]!==undefined&&arguments[0]!==null && typeof arguments[0]==="string"&&(arguments[0]=arguments[0].trim())!==""){
                //_flush();
                target=arguments[0];
            }
        }
        if (template!==undefined&&template!==null) {
            conttentprepped=Array.isArray(template)?template.join(""):template;
            var trgti=-1;
            var lstTrgti=-1;
            var crntcntnttowrite=""
            while(conttentprepped.length>0){
                if((trgti=conttentprepped.indexOf("[#trgt#"))>-1){
                    lstTrgti=trgti;
                    var precntnt=conttentprepped.substring(trgti);
                    conttentprepped=conttentprepped.substring(trgti,"[#trgt#".length)
                    if (precntnt===undefined||precntnt===null){
                        precntnt="";
                    }
                    crntcntnttowrite+=precntnt;
                    if ((trgti=conttentprepped.indexOf("#]"))>-1){
                        var trgtnme=conttentprepped.substring(trgti);
                        conttentprepped=conttentprepped.substring(trgti,"#]".length)
                        if(trgtnme===undefined||trgtnme===null){
                            trgtnme="";
                        }
                        if((trgti=conttentprepped.indexOf(`[#trgt#${trgtnme}#]`))>-1) {
                            if (crntcntnttowrite!==undefined&&crntcntnttowrite!==null&&crntcntnttowrite!==""){
                                write(crntcntnttowrite);
                                crntcntnttowrite="";
                            }
                            _target(trgtnme);
                            crntcntnttowrite= conttentprepped.substring(0,trgti);
                            write(crntcntnttowrite);
                            crntcntnttowrite="";
                            conttentprepped=conttentprepped.substring(trgti,`[#trgt#${trgtnme}#]`.length);
                        }
                    }
                } else {
                    crntcntnttowrite+=conttentprepped;
                    write(crntcntnttowrite);
                    crntcntnttowrite="";
                    break;
                }
            }
        }
    }
}