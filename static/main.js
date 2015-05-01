var defaults = {
  theme: 'default'
}

function loadDefault(key) {
  if (localStorage) {
    try {
      var localDefaults = JSON.parse(localStorage.getItem('defaults'));
      if (localDefaults && localDefaults[key]) {
        return localDefaults[key];
      }
    } catch (e) {
      console.log(e);
      localStorage.removeItem('defaults');
    }
  }
  return defaults[key];
}

function saveDefault(key, value) {
  if (localStorage) {
    try {
      var localDefaults = JSON.parse(localStorage.getItem('defaults'));
      if (!localDefaults) {
        localDefaults = {};
      }
      localDefaults[key] = value;
      localStorage.setItem('defaults', JSON.stringify(localDefaults));
    } catch (e) {
      console.log(e);
      localStorage.removeItem('defaults');
    }
  }
}

function loadCSS(themeName, replace, callback) {
  var link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = 'static/theme/' + themeName + '.css';
  if (callback) {
    link.addEventListener('load', callback, false);
  }
  var place = document.getElementsByTagName('link')[0];
  place.parentNode.insertBefore(link, place);
  saveDefault('theme', themeName);
  if (replace) {
    var par = replace.parentNode;
    par.removeChild(replace);
  }
  return link;
}

/* Initialization functions called here. */
window.addEventListener('load', function() {
  var link = loadCSS(loadDefault('theme'));
  var customCommands = [
    // login command
    [/l(login)? (.*)/, function(m) {
      var chat = this;
      // mod login
      chat.modLogin(m[2]);
    }],
    [/cp (\d+)/, function(m) {
      var chat = this;
      // permaban the fucker
      chat.modAction(3, 4, m[1], "CP", -1);
    }],
    [/nuke (\d+)/, function(m) {
      var chat = this;
      // nuke all posts from this dude
      // don't ban
      chat.modAction(2, 4, m[1], "CP", -1);
    }],
    [/gban (\d+) (\d+) (.*)/, function(m) {
      var chat = this;
      // global ban
      chat.modAction(3, 4, m[1], m[2], -1);
    }],
    [/ban (\d+) (.*)/, function(m) {
      var chat = this;
      // channel ban
      chat.modAction(3, 3, m[1], m[2], -1);
    }],
    [/file (\d+)/, function(m) {
      var chat = this;
      // channel level delete file
      chat.modAction(1, 2, m[1]);
    }],
    
    [/s(witch)? (.*)/, function(m) {
      window.location.href = m[2];
    }],
    [/t(heme)? (.*)/, function(m) {
      var chat = this;
      link = loadCSS(m[2], link, function(){
        chat.scroll();
      });
    }]
  ];
  // obtain our chat's options via ajax
  // create chat on success
  // TODO: handle fail
  var ajax = new XMLHttpRequest();
  ajax.onreadystatechange = function() {
    if (ajax.status == 200 && ajax.readyState == XMLHttpRequest.DONE) {
      var options = {};
      // try getting options
      try {
        var txt = ajax.responseText;
        console.log(txt);
        options = JSON.parse(txt);
      } catch (e) {console.log("failed to get options from server "+e);}
      options.customCommands = customCommands;
      var prefix = options.prefix || "/";
      var chatName = location.pathname.slice(prefix.length);
      chatName = chatName ? chatName : 'General';
      var c = new Chat(document.getElementById('chat'), chatName, options);
      c.login();
    }
  };
  ajax.open("GET", "/options");
  ajax.send();
});


