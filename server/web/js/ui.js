

function messageTypeToggled(id) {
    var checkBox = document.getElementById(id);
    console.log('xhr request to set ' + id + ' to ' + checkBox.checked);

    fetch('http://localhost:8082/filters/toggle?requestType=' + id + '&enabled=' + checkBox.checked)
    .then(response => {
      console.log(response);
      if (!response.ok) {
        alert('Failed to set filter ' + id + ' to state ' + checkBox.checked);
        checkBox.checked = !checkBox.checked;
      }
    }).catch(err => {
      console.log(err);
      alert('Failed to set filter ' + id + ' to state ' + checkBox.checked + ', error: ' + err);
      checkBox.checked = !checkBox.checked;
    });
}//https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch

function flushDB() {
  flush('redis');
}

function flushMsgs() {
  flush('messages');
}

function flush(type) {
  console.log('Flushing ' + type);
  var currentTime = +new Date();
  console.log(currentTime);

  fetch('http://localhost:8082/' + type + '/flush?epoch=' + currentTime)
    .then(response => {
      console.log(response);
      if (!response.ok) {
        alert('Failed to flush ' + type);
      }
    }).catch(err => {
      console.log(err);
      alert('Failed to flush ' + type);
    });
}
