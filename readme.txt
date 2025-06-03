Testing/Development
===================

$ make run

Will build and run a local copy with example config
from config.json.  That should give you a demo.
Tweak the values in config.json in case you have port
conflicts.

$ ./switchboard-web-ui -register

Will register a user.  This gives you a registration
URL.  Visiting the URL sets a cookie.
After that you are authenticated for your browser and 
device.  The only purpose of the registration is to 
avoid someone reading the URL and trying it on their
device.  For better security tokens should be 1-time
use but this is enough for me.


Installation
============

Although the code runs on MacOs install is only 
suported on Linux systemd.

$ make install

Will setup the system with some default locations. 
Install files manually if you want to change default locations.
The locations for the various directories are hard-coded
into the binary.  You can override them with the config file.
Anything missing from the config file will fall back 
to the defaults in main.go.

 