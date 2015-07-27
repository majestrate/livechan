--
-- postgres.sql
-- postgres backend database schema
--

-- begin tables

CREATE TABLE IF NOT EXISTS Users(        -- table for registered users
  userName VARCHAR(64) PRIMARY KEY,      -- the username for this user
  userLogin VARCHAR(64) NOT NULL,        -- the login username used for login
  userHash TEXT NOT NULL,                -- the password hash for login
  userSalt TEXT NOT NULL,                -- the salt used in the password hash
  created INTEGER NOT NULL,              -- when this user was created
  lastLogin INTEGER NOT NULL DEFAULT -1  -- when this user last logged in, or -1 for never
);

CREATE TABLE IF NOT EXISTS Boards(                                -- Boards table, holds entries for boards
  channelName VARCHAR(32) PRIMARY KEY,                            -- the name of the channel
  created INTEGER NOT NULL,                                       -- when it was created
  slogan TEXT NOT NULL,                                           -- channel slogan
  description TEXT NOT NULL,                                      -- channel description
  owner VARCHAR(64) REFERENCES Users(userName) ON DELETE CASCADE  -- the name of the owner
);

CREATE TABLE IF NOT EXISTS Convos(       -- converstation
  convoID SERIAL PRIMARY KEY,            -- convo id
  convoName VARCHAR(128),                -- the name of this convo
  channelName VARCHAR(32),               -- which channel this convo is in 
  lastBump INTENGER NOT NULL,            -- the time at which someone bumped this convo last
  created INTEGER NOT NULL DEFAULT now() -- when this convo was created
);

CREATE TABLE IF NOT EXISTS Chats(                                       -- Posts on a board
  postID SERIAL PRIMARY KEY,                                            -- post number or id
  boardName VARCHAR(32) REFERENCES Boards(channelName),                 -- the board this was on
  postName VARCHAR(128) NOT NULL DEFAULT "Anonymous",                   -- name of a poster
  postTrip VARCHAR(32),                                                 -- poster's tripcode
  postCountry VARCHAR(8),                                               -- country code of poster
  postMessage TEXT,                                                     -- the message contents of the post
  postTime INTEGER NOT NULL DEFAULT now(),                              -- when this was posted
  postConvo INTEGER REFERENCES Convos(convoID) ON DELETE CASCADE,       -- the converstation this post belongs to
  postAddr VARCHAR(64) NOT NULL,                                        -- the ip address of the poster
  postFilename
);

-- end tables
