var socket;
var mode;
var needsBase64;
const messages = [];

function intiializeWS(whichMode) {
    // socket = new WebSocket("ws://localhost:8082/filters/ws");
    console.log("Attempting Connection...");
    if (window.location.protocol.includes("https")) {
        console.log("Attempting wss..");
        socket = new WebSocket("wss://" + location.host + "/filters/ws");
    } else {
        socket = new WebSocket("ws://" + location.host + "/filters/ws");
    }
    /*try {
        socket = new WebSocket("ws://" + location.host + "/filters/ws");
    } catch (error) {
        console.log("WebSocket connect error: " + error);
        console.log("Attempting wss..");
        socket = new WebSocket("wss://" + location.host + "/filters/ws");
    }*/

    mode = whichMode;

    socket.onopen = () => {
        console.log("Successfully Connected");
        // socket.send('{"Hi": "From the Client!"}')
        wsPing();
    };

    socket.onmessage = onMessage;
    socket.onclose = onClose;
    socket.onerror = onError;
    if (mode === 'proxy') {
        document.getElementById('proxySubmitMsg').disabled = true;
    } else {
        document.getElementById('submitMsg').disabled = true;
    }
}

function onMessage(event) {
    //https://developer.mozilla.org/en-US/docs/Web/API/HTMLTextAreaElement
    //
    console.log('message received')
    console.log(event.data);
    messages.push(event.data);
    console.log('mode: ' + mode);
    if (messages.length === 1) {
        if (mode === "proxy") {
            displayNextProxyMessage();
        } else {
            displayNextMessage();
        }
    }
}

function onClose(event) {
    console.log("Socket Closed Connection: ", event);
    socket.send("Client Closed!")
}

function onError(error) {
    console.log("Socket Error: ", error);
}

function sendMessage(msg) {
    socket.send(msg);
}

function displayNextMessage() {
    // //var data = messages.shift();
    // var data = messages[0];
    // var obj = JSON.parse(data);
    // var uuid = obj.uuid;
    // var reqType = obj.requestType;
    // console.log(uuid + " : " + reqType);
    // delete obj.uuid;
    // delete obj.requestType;
    // var str = JSON.stringify(obj, undefined, 4);
    var resp = dequeueNextMessage();
    document.getElementById('messageArea').value = JSON.stringify(resp.message, undefined, 4);
    document.getElementById('uuid').value = resp.uuid;
    document.getElementById('reqType').innerText = resp.type;
    document.getElementById('submitMsg').disabled = false;
}

function displayNextProxyMessage() {
    var resp = dequeueNextMessage();
    /*
        if resp.message.base64Content content is non-json, base64 decode
    */
    if (resp.message.base64Content) {
        document.getElementById('proxyMessageArea').value = Base64.decode(resp.message.base64Content);
        needsBase64 = true;
    } else {
        //document.getElementById('proxyMessageArea').value = JSON.stringify(resp.message, undefined, 4);
    }
    document.getElementById('proxyMessageHeaders').value = JSON.stringify(resp.message.httpHeaders, undefined, 4);
    // document.getElementById('proxyMessageArea').value = resp.message;
    // document.getElementById('proxyMessageArea').value = JSON.stringify(resp.message, undefined, 4);
    document.getElementById('proxyUuid').value = resp.uuid;
    document.getElementById('proxyReqType').innerText = resp.type;
    document.getElementById('proxySubmitMsg').disabled = false;
}

function dequeueNextMessage() {
    needsBase64 = false;
    // var data = messages.shift();
    var data = messages[0];
    var obj = JSON.parse(data);
    var uuid = obj.uuid;
    var reqType = obj.requestType;
    console.log(uuid + " : " + reqType);
    delete obj.uuid;
    delete obj.requestType;
    // var str = JSON.stringify(obj, undefined, 4);
    // return {message: str, uuid: uuid, type: reqType};
    return {message: obj, uuid: uuid, type: reqType};
}

function submitMessage() {
    // document.getElementById('submitMsg').disabled = true;
    // var uuid = document.getElementById('uuid').value;
    // var msg = document.getElementById('messageArea').value;
    // var str;
    // try {
    //     var obj = JSON.parse(msg);
    //     obj["uuid"] = uuid;
    //     str = JSON.stringify(obj);
    // } catch (error) {
    //     console.log('Error with Json: ' + error);
    //     alert('Error with Json: ' + error);
    //     document.getElementById('submitMsg').disabled = false;
    //     return;
    // }
    // document.getElementById('messageArea').value = "";
    // document.getElementById('uuid').value = "";
    // document.getElementById('reqType').innerText = "";
    // sendMessage(str);
    // messages.shift();
    // if (messages.length > 0) {
    //     displayNextMessage();
    // }

    var str = parseMessage(document.getElementById('submitMsg'), document.getElementById('messageArea'), document.getElementById('uuid'));
    if (!str) {
        return;
    }
    document.getElementById('reqType').innerText = "";
    sendMessage(str);
    messages.shift();
    if (messages.length > 0) {
        //displayNextMessage();
        if (mode === "proxy") {
            displayNextProxyMessage();
        } else {
            displayNextMessage();
        }
    }
}

function submitProxyMessage() {
    var str = parseMessage(document.getElementById('proxySubmitMsg'), document.getElementById('proxyMessageArea'), document.getElementById('proxyUuid'), document.getElementById('proxyMessageHeaders'));
    if (!str) {
        return;
    }
    document.getElementById('proxyReqType').innerText = "";
    sendMessage(str);
    messages.shift();
    if (messages.length > 0) {
        displayNextProxyMessage();
    }
}

function parseMessage(submitElement, messageElement, uuidElement, httpHeaderElement) {
    submitElement.disabled = true;
    var uuid = uuidElement.value;
    console.log('uuid: ' + uuid)
    var msg = messageElement.value;
    if (needsBase64) {
        msg = `{"base64Content": "${Base64.encode(msg)}"}`
    }
    console.log(msg)
    var str;
    try {
        var obj; // = JSON.parse(msg);
        if (msg !== undefined && msg !== '') {
            obj = JSON.parse(msg);
        } else {
            obj = JSON.parse('{}');
        }
        obj["uuid"] = uuid;
        
        if (httpHeaderElement !== undefined && httpHeaderElement.value !== undefined && httpHeaderElement.value !== '') {
            const headers = JSON.parse(httpHeaderElement.value);
            obj['httpHeaders'] = headers;
        }
        
        str = JSON.stringify(obj);
        console.log('str: ' + str)
    } catch (error) {
        console.log('Error with Json: ' + error);
        alert('Error with Json: ' + error);
        submitElement.disabled = false;
        return;
    }
    messageElement.value = "";
    uuidElement.value = "";
    if (httpHeaderElement !== undefined) {
        httpHeaderElement.value = "";
    }
    // document.getElementById('proxyReqType').innerText = "";
    return str;
}

function wsPing() {
    if (!socket) return;
    if (socket.readyState !== 1) return;
    sendMessage('{"uuid": "ping"}');
    setTimeout(wsPing, 10000);
}