# The software renderer in this folder

Mesa 3D, version `26.2.5`. MIT AND Apache-2.0 WITH LLVM-exception AND NCSA AND BSL-1.0 AND BSD-3-Clause.

OpenGL drawn by the processor instead of a graphics card, for the window of QA
Data Generator on Windows. It is not part of `qadg-gui.exe`. It is two files in
a folder called `opengl` beside it:

- `opengl/libgallium_wgl.dll`, 61880320 bytes, SHA-256 `700e1fb7b9487d9d1c53752b03aeb040d22f3731b46aa254d88830b6279474de`
- `opengl/opengl32.dll`, 139264 bytes, SHA-256 `c8511461ba71a0288750467e3be2f5c1c2bbbceaaf0ab5d8c496b0ba7302daa8`

The window loads them only when the graphics driver gives it no OpenGL, as in
a virtual machine without 3D acceleration, over a remote desktop or on a
server, or when it is started with `-software-gl`. It then starts a second
time and draws everything with the processor, which is slower. On a computer
whose driver draws the window the two files are never opened, and deleting
this folder changes nothing there.

## Where they come from

They are `x64/libgallium_wgl.dll` and `x64/opengl32.dll` out of
`mesa3d-26.2.5-release-msvc.7z`, SHA-256 `997123ef4ce8896204ab912d5b54f19372c487ff9835bcefafa2fca23b8953c2`, which
pal1000/mesa-dist-win builds from the source of Mesa and publishes at
<https://github.com/pal1000/mesa-dist-win/releases/tag/26.2.5>. The sizes and
sums above are those of the files as that archive holds them, so the files
here can be compared with that archive without trusting anybody in between.

In the repository of QA Data Generator this folder is `.github/opengl`, with
the notices and without the two files. `.github/scripts/fetch_software_renderer.sh`
downloads the archive, refuses it unless its sum is the one above, takes the
two files out, checks each one, and puts them here together with these notices.

## Licences

The archive carries no licence file. What is compiled into the two files was
read from the files themselves, which name Mesa 26.2.5 and LLVM 23.1.4, and the
texts in this folder are taken from the source of those two versions and of
the Direct3D 12 headers at the version the build information of that release
names, 1.619.5, in `buildinfo/msvc.txt` at the tag. That build information also
names zlib and zstd, and neither is compiled into the files: the only mention
of either is a message of LLVM saying it was built without them. The second
file imports the first, and beyond that they import libraries of Windows itself
and no other.

| File in this folder | What it holds | Licence |
|---|---|---|
| `LICENSE-Mesa.txt` | the notices of the core library, of each driver compiled in, and of the shader texts kept in the file | MIT, BSD-3-Clause |
| `LICENSE-BSL-1.0.txt` | the licence of the C11 threads library, which Mesa compiles on Windows in place of `<threads.h>`, as Mesa keeps it in `licenses/BSL-1.0` | BSL-1.0 |
| `LICENSE-LLVM.txt` | `llvm/LICENSE.TXT` at the tag `llvmorg-23.1.4`, whole. LLVM is what turns shaders into machine code. The text ends with LLVM's older licence, which LLVM says still covers a small part of its code alone | Apache-2.0 WITH LLVM-exception, NCSA |
| `LICENSE-DirectX-Headers.txt` | `LICENSE` of the Direct3D 12 headers at the tag `v1.619.5`, whole | MIT |
