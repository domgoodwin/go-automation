import sys, plotly

if len(sys.argv) != 3:
    print("Need 2 args (in, out)")
    exit(2)


in_file = sys.argv[1]
out_file = sys.argv[2]

f = open(in_file, "r")
fig = plotly.io.from_json(f.read())
fig.update_layout(autotypenumbers='convert types')
fig.write_image(out_file)

exit(0)