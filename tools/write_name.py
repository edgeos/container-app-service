import sys
import os

text = bytearray(bytes(sys.argv[2], 'utf-8'))
zero = bytearray([0])
with open(sys.argv[1], "wb+") as f:
    f.write(text)
    f.write(zero)
    f.close()
