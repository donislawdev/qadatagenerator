# Media

The animations the front page of the repository plays.

## What is here

| File | What it is |
|---|---|
| `qa-in-action.webp` | A looping animation of the window, 20 seconds long, 1200 by 836 pixels at 20 frames per second, 1 MB. On Quick value it opens the list of kinds and makes ten test card numbers. On Damage a file it picks kinds of damage and opens more options. On Recipe it scrolls through the example recipe. |
| `qa-star.gif` | A looping animation of the top of this repository's page on GitHub, 2.3 seconds long, 1852 by 66 pixels, 30 KB. The pointer clicks Star and the button turns into Starred. |

## Why the window is an animation and not a film

GitHub plays a film on a page as a player when the film was uploaded through
its own editor or an issue. The upload keeps the file outside the repository,
at an address on the `github.com/user-attachments` path. A looping picture kept
here is shown everywhere and stays with the repository, so the front page plays
this one.

The film it was made from is not kept here, because a file in the history of a
repository stays there for good. It was a 20 second recording of 48 MB, 1680 by
1170 pixels at 60 frames per second, and the animation was made from it with
FFmpeg:

```
ffmpeg -i qa-in-action.mp4 -vf "fps=20,scale=1200:-1:flags=lanczos" -c:v libwebp_anim -lossless 0 -q:v 80 -compression_level 6 -loop 0 qa-in-action.webp
```

## Changing it

Replace a file and keep its name, so the README does not change. Every version
stays in the history of the repository for good, so a new animation should be
no larger than the one it replaces.

Look at an animation before committing it. What is on screen goes public with
it: window titles, file names and paths.
