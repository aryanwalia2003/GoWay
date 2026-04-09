mkdir -p /tmp/tectonic-test
cd /tmp/tectonic-test
cat << 'EOF' > preamble.tex
\documentclass{article}
\usepackage{geometry}
EOF

cat << 'EOF' > macros.tex
\def\customerName{Test User}
EOF

cat << 'EOF' > body.tex
\begin{document}
Hello \customerName
\end{document}
EOF

cat << 'EOF' > Tectonic.toml
[doc]
name = "texput"
inputs = ["preamble.tex", "macros.tex", "body.tex"]
EOF

# Add the path to local tectonic binary if needed or assume global is available
/home/aryanwalia/personal/GoWay/tectonic -X build
find . -name "*.pdf"
