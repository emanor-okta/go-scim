

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

function logMessagesToggled() {
  var checkBox = document.getElementById('LogMessages');

  fetch('http://localhost:8082/messages/toggle?enabled=' + checkBox.checked)
    .then(response => {
      console.log(response);
      if (!response.ok) {
        alert('Failed to change message logging to ' + checkBox.checked);
        checkBox.checked = !checkBox.checked;
      }
    }).catch(err => {
      console.log(err);
      alert('Failed to change message logging to ' + checkBox.checked + ', error: ' + err);
      checkBox.checked = !checkBox.checked;
    });
}

function updateUser(id) {
  var userId = document.getElementById('id-'+id).value;
  var msg = document.getElementById('messageArea-'+id).value;
  sendPost("http://localhost:8082/users/update?id=" + userId, msg);
}

function deleteUser(id) {
  var userId = document.getElementById('id-'+id).value;
  fetch('http://localhost:8082/users/delete?id=' + userId)
  .then(response => {
    console.log(response);
    if (!response.ok) {
      alert('Failed to delete user ' + response.statusText);
      return;
    }
    window.location.reload();
  }).catch(err => {
    console.log(err);
    alert('Failed to delete user ' + err);
  });
}

function updateGroup(id) {
  var groupId = document.getElementById('id-'+id).value;
  var msg = document.getElementById('messageArea-'+id).value;
  sendPost("http://localhost:8082/groups/update?id=" + groupId, msg);
}

function deleteGroup(id) {
  var groupId = document.getElementById('id-'+id).value;
  fetch('http://localhost:8082/groups/delete?id=' + groupId)
  .then(response => {
    console.log(response);
    if (!response.ok) {
      alert('Failed to delete group ' + response.statusText);
      return;
    }
    window.location.reload();
  }).catch(err => {
    console.log(err);
    alert('Failed to delete group ' + err);
  });
}

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

function sendPost(url, msg) {
  const options = {
    method: "POST",
    body: msg,
  };
  const request = new Request(url, options);
  fetch(request)
  .then(response => {
    console.log(response);
    if (!response.ok) {
      alert('Failed posting to: ' + url +', status: ' + response.statusText);
    }
  }).catch(err => {
    console.log(err);
    alert('Failed posting to: ' + url +', error: ' + err);
  });
}
