import sys
import os

byte = abs(os.path.getsize(sys.argv[1]) - os.path.getsize(sys.argv[2])) % 16
# add salt
bytes = bytearray(os.urandom(16))
# add padding
bytes.extend(bytearray([byte]))
with open(sys.argv[3], "wb+") as f:
    f.write(bytes)
    f.close()
