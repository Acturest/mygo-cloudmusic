# 獅尾四季春加糖SC

`SweiSpringSugarCJKsc-Regular.ttf` is a subset of Swei Spring Sugar CJK SC
Regular, version 1.061, from https://github.com/max32002/swei-spring
(`CJK SC/SweiSpringSugarCJKsc-Regular.ttf`, 26,502,228 bytes).

It keeps the characters of GB 2312 and JIS X 0208, the kana, Latin and
punctuation, and drops the hinting: 10,793 characters, 6,064,728 bytes.
Anything else falls back to the system's fonts. To remake it:

    pyftsubset swei-spring/"CJK SC/SweiSpringSugarCJKsc-Regular.ttf" \
        --text-file=chars.txt --no-hinting --drop-tables+=FFTM \
        --output-file=SweiSpringSugarCJKsc-Regular.ttf

The font is under the SIL Open Font License 1.1, as `OFL.txt` says.
