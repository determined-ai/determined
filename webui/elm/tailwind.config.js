module.exports = {
  variants: {
    backgroundColor: ['hover', 'group-hover'],
    borderColor: ['hover', 'group-hover'],
    cursor: ['disabled'],
    opacity: ['disabled', 'hover'],
    textColor: ['hover', 'group-hover'],
    visibility: ['hover', 'group-hover'],
    zIndex: ['focus'],
  },
  theme: {
    lineHeight: {
      inherit: 'inherit !important',
    },
    extend: {
      padding: {
        '1-5': '0.375rem',
      },
      boxShadow: {
        'outline-orange': '0 0 0 3px #fbd38d',
      },
    },
  },
}
