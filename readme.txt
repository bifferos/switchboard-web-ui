Testing/Development
===================

$ make run

Will build and run a local copy with example config
from config.json.  That should give you a demo.
Tweak the values in config.json in case you have port
conflicts.

$ make register

Will register a user.  This gives you some html in a 
file called ./register.html.  Just open it in the browser 
and either follow the link (in the browser) or get
your mobile out and read the QR code.  

This will register a cookie on your device for access.
The cookie never expires.

Installation
============

Although the code runs on MacOs install is only 
suported on Linux systemd.

$ make install

Will setup the system with some default locations. 
Check the makefile, and tweak of you don't like where it puts
things.  If you change any entries in the makefile be
sure to also change in /etc/switchboard-web-ui/config.json
see config.json for an example.  Normally the etc config.json
is not required because the locations are hard-coded into the
binary.

$ make uninstall

Will remove files, but be warned it also removes widget 
files you may have setup, and state so back up if you've hand
crafted something complicated.
 