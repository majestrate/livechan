var modCommands = [
  // login command
  [/l(login)? (.*)/, function(m) {
    var chat = this;
    // mod login
      chat.modLogin(m[2]);
  },
   "login as user", "/l user:password",
  ],
  [/cp (\d+)/, function(m) {
    var chat = this;
    // permaban the fucker
    chat.modAction(3, 4, m[1], "CP", -1);
  },
   "handle illegal content", "/cp postnum",
  ],
  [/cnuke (\d+) (.*)/, function(m) {
    var chat = this;
    // channel ban + nuke files
    chat.modAction(2, 4, m[1], m[2], -1);
  },
   "channel level ban+nuke", "/cnuke postnum reason goes here",
  ],
  [/purge (\d+) (.*)/, function(m) {
    var chat = this;
    // channel ban + nuke files
    chat.modAction(2, 9, m[1], m[2], -1);
  },
   "channel level ban+nuke", "/cnuke postnum reason goes here",
  ],
  [/gnuke (\d+) (.*)/, function(m) {
    var chat = this;
    // global ban + nuke with reason
    chat.modAction(3, 4, m[1], m[2], -1);
  },
   "global ban+nuke", "/gnuke postnum reason goes here",
  ],
  [/gban (\d+) (.*)/, function(m) {
    var chat = this;
    // global ban with reason
    chat.modAction(3, 3, m[1], m[2], -1);
  },
   "global ban (no nuke)", "/gban postnum reason goes here",
  ],
  [/cban (\d+) (.*)/, function(m) {
    var chat = this;
    // channel ban with reason
    chat.modAction(2, 3, m[1], m[2], -1);
  },
   "channel level ban (no nuke)", "/cban postnum reason goes here",
  ],
  [/dpost (\d+)/, function(m) {
    var chat = this;
    // channel level delete post
    chat.modAction(1, 2, m[1]);
  },
   "delete post and file", "/dpost postnum",
  ],
  [/dfile (\d+)/, function(m) {
    var chat = this;
    // channel level delete file
    chat.modAction(1, 1, m[1]);
  },
   "delete just file", "/dpost postnum",
  ]
]
