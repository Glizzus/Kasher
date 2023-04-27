from subprocess import run
from os import environ, chdir, getcwd

owd = getcwd()


def compile_go(type, arch, os):
    print(f'Compiling {type} for architecture {arch} for OS {os}')
    chdir(type)
    binary = f"../bin/kasher{type}-{arch}-{os}.exe"
    source = '.'
    run(["go", "build", "-ldflags", "-s -w", "-o", binary, source])


for os in ["linux", "windows", "darwin"]:
    environ["GOOS"] = os
    for arch in ["amd64"]:
        environ["GOARCH"] = arch
        for type in ["client", "server"]:
            compile_go(type, arch, os)
            chdir(owd)
