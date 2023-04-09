from subprocess import run
from os import environ


def compile_go(prog, arch, os):
    print(f'Compiling {prog} for architecture {arch} for OS {os}')
    binary = f"bin/kasher{prog}-{arch}-{os}.exe"
    source = f"{prog}/{prog}.go"
    run(["go", "build", "-ldflags", "-s -w", "-o", binary, source])


for os in ["linux", "windows", "darwin"]:
    environ["GOOS"] = os
    for arch in ["amd64"]:
        environ["GOARCH"] = arch
        for prog in ["client", "server"]:
            compile_go(prog, arch, os)
