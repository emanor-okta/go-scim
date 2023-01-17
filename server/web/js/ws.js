var socket;
const messages = [];

function intiializeWS() {
    socket = new WebSocket("ws://localhost:8082/filters/ws");
    console.log("Attempting Connection...");

    socket.onopen = () => {
        console.log("Successfully Connected");
        // socket.send('{"Hi": "From the Client!"}')
    };

    socket.onmessage = onMessage;
    socket.onclose = onClose;
    socket.onerror = onError;
    document.getElementById('submitMsg').disabled = true;
}

function onMessage(event) {
    //https://developer.mozilla.org/en-US/docs/Web/API/HTMLTextAreaElement
    //
    console.log('message received')
    console.log(event.data);
    messages.push(event.data);
    if (messages.length === 1) {
        displayNextMessage();
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
    // var data = messages.shift();
    var data = messages[0];
    var obj = JSON.parse(data);
    var uuid = obj.uuid;
    var reqType = obj.requestType;
    console.log(uuid + " : " + reqType);
    delete obj.uuid;
    delete obj.requestType;
    var str = JSON.stringify(obj, undefined, 4);
    document.getElementById('messageArea').value = str;
    document.getElementById('uuid').value = uuid;
    document.getElementById('reqType').innerText = reqType;
    document.getElementById('submitMsg').disabled = false;
}

function submitMessage() {
    document.getElementById('submitMsg').disabled = true;
    var uuid = document.getElementById('uuid').value;
    var msg = document.getElementById('messageArea').value;
    var obj = JSON.parse(msg);
    obj["uuid"] = uuid;
    var str = JSON.stringify(obj);
    document.getElementById('messageArea').value = "";
    document.getElementById('uuid').value = "";
    document.getElementById('reqType').innerText = "";
    sendMessage(str);
    messages.shift();
    if (messages.length > 0) {
        displayNextMessage();
    }
}