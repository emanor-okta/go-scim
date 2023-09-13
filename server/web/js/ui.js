
function messageTypeToggled(id) {
    var checkBox = document.getElementById(id);
    console.log('xhr request to set ' + id + ' to ' + checkBox.checked);

    // fetch('http://localhost:8082/filters/toggle?requestType=' + id + '&enabled=' + checkBox.checked)
    fetch(location.origin + '/filters/toggle?requestType=' + id + '&enabled=' + checkBox.checked)
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
  // if (checkBox.checked && document.getElementById('ProxyLogMessages').checked) {
  //   alert('Failed to enable message logging, Proxy logging already enabled');
  //   checkBox.checked = false;
  //   return;
  // }


  // fetch('http://localhost:8082/messages/toggle?enabled=' + checkBox.checked)
    fetch(location.origin + '/messages/toggle?enabled=' + checkBox.checked)
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

function logProxyMessagesToggled() {
  var checkBox = document.getElementById('ProxyLogMessages');
  // if (checkBox.checked && document.getElementById('LogMessages').checked) {
  //   alert('Failed to enable Proxy logging, Message logging already enabled');
  //   checkBox.checked = false;
  //   return;
  // }

  var port = document.getElementById('Port');
  var origin = document.getElementById('Origin');
  if (checkBox.checked && (port.value === '' || origin.value === '')) {
    alert('proxy_port and origin_url can\'t be blank');
    checkBox.checked = false;
    return;
  }

  // fetch('http://localhost:8082/proxy/toggle?enabled='+checkBox.checked+'&url='+origin.value+'&port='+port.value)
  fetch(location.origin + '/proxy/toggle?enabled='+checkBox.checked+'&url='+origin.value+'&port='+port.value)
  .then(response => {
    console.log(response);
    if (!response.ok) {
      response.json()
      .then(data => {
        alert('Failed to change proxy logging to ' + checkBox.checked + '\nError: ' + data["error"]);
      });
      checkBox.checked = !checkBox.checked;
    } else if (checkBox.checked) {
      port.disabled = true;
      origin.disabled = true;
    } else {
      port.disabled = false;
      origin.disabled = false;
    }
  }).catch(err => {
    console.log(err);
    alert('Failed to change proxy logging to ' + checkBox.checked + ', error: ' + err);
    checkBox.checked = !checkBox.checked;
  });
}

function updateUser(id) {
  var userId = document.getElementById('id-'+id).value;
  var msg = document.getElementById('messageArea-'+id).value;
  sendPost(location.origin + "/users/update?id=" + userId, msg, success);
  // sendPost("http://localhost:8082/users/update?id=" + userId, msg, success);
}

function createNewUser() {
  const user = document.getElementById("newUserArea").value;
  sendPost(location.origin + "/users/update", user, success);
}

function deleteUser(id) {
  var userId = document.getElementById('id-'+id).value;
  fetch(location.origin + '/users/delete?id=' + userId)
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
  sendPost(location.origin + "/groups/update?id=" + groupId, msg, success);
}

function createNewGroup() {
  const group = document.getElementById("newGroupArea").value;
  sendPost(location.origin + "/groups/update", group, success);
}

function deleteGroup(id) {
  var groupId = document.getElementById('id-'+id).value;
  fetch(location.origin + '/groups/delete?id=' + groupId)
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
  fetchTemplateData(location.origin + '/raw/user.json', document.getElementById("newUserArea"));
}

function populateNewGroup() {
  fetchTemplateData(location.origin + '/raw/group.json', document.getElementById("newGroupArea"));
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

  fetch(location.origin + '/' + type + '/flush?epoch=' + currentTime)
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

var refreshInterval = 0;
function refreshMessages(event) {
  console.log(event);
  if (this.value == "Auto Refresh Off") {
    console.log('Auto Refresh Off');
    refreshInterval = 0;
  } else if (this.value == "5") {
     console.log('5');
     refreshInterval = 5000;
  } else if (this.value == "15") {
    console.log('15');
    refreshInterval = 15000;
  } else if (this.value == "30") {
    console.log('30');
    refreshInterval = 30000;
  } else if (this.value == "60") {
    console.log('60');
    refreshInterval = 60000;
  }

  sessionStorage.setItem('refreshInterval', refreshInterval.toString());
  if (refreshInterval > 0) {
    startRefresh();
  }
}

function startRefresh() {
  console.log('start timeout');
  refreshInterval = parseInt(sessionStorage.getItem('refreshInterval'));
  var index;
  if (refreshInterval == 5000) {
    index = 1;
  } else if (refreshInterval == 15000) {
    index = 2;
  } else if (refreshInterval == 30000) {
    index = 3;
  } else if (refreshInterval == 60000) {
    index = 4;
  } else {
    index = 0;
  }

  document.getElementById('AutoRefresh').selectedIndex = index;
    
  setTimeout(() => {
    console.log('awak-' + refreshInterval);
    if (refreshInterval > 0) {
      window.location.reload();
    } 
  }, refreshInterval);
}

// document.getElementById('AutoRefreshProxyMessages').addEventListener("change", refreshMessages);
// document.getElementById('AutoRefresh').addEventListener("change", refreshMessages);