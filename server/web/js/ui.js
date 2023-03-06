

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
  sendPost("http://localhost:8082/users/update?id=" + userId, msg, success);
}

function createNewUser() {
  const user = document.getElementById("newUserArea").value;
  sendPost("http://localhost:8082/users/update", user, success);
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
  sendPost("http://localhost:8082/groups/update?id=" + groupId, msg, success);
}

function createNewGroup() {
  const group = document.getElementById("newGroupArea").value;
  sendPost("http://localhost:8082/groups/update", group, success);
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

function populateNewUser() {
  fetchTemplateData('http://localhost:8082/raw/user.json', document.getElementById("newUserArea"));
}

function populateNewGroup() {
  fetchTemplateData('http://localhost:8082/raw/group.json', document.getElementById("newGroupArea"));
}

function fetchTemplateData(url, element) {
  fetch(url)
  .then(response => {
    if (!response.ok) {
      alert('Failed to load content from: ' + url + '\nStatus: ' + response.statusText);
      return;
    }
    response.json()
    .then(data => {
      console.log(JSON.stringify(data, null, "  "));
      element.textContent = JSON.stringify(data, null, "  ");
    });
  }).catch(err => {
    console.log(err);
    alert('Failed to load content from: ' + url + '\nError: ' + err);
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

function sendPost(url, msg, success) {
  const options = {
    method: "POST",
    body: msg,
  };
  const request = new Request(url, options);
  fetch(request)
  .then(response => {
    console.log(response);
    if (!response.ok) {
      response.text()
      .then(d => {
        alert('Failed posting to: ' + url +'\nStatus: ' + response.statusText + '\nError: ' + d);
      });
      return;
    }
    success();
  }).catch(err => {
    console.log(err);
    alert('Failed posting to: ' + url +', error: ' + err);
  });
}

function success() {
  window.location.reload();
}
