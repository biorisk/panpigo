panpigo
=======

Control pianobar on a Raspberry Pi using webserver written in golang.

Pianobar (https://github.com/PromyLOPh/pianobar) is a great way to
listen to a popular internet radio station.  I have been using it
on my Raspberry Pi, but wanted a headless mode. Originally, I
created a webserver to control pianobar in Perl, but ran into
stability issues (my fault, not Perl's).  Go has interested me for
several months, so this provided a good test project. I threw in
websockets for the heck of it.

Screen shot of browser:
![Alt text](/screenshot.png "screenshot")
