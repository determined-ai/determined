import React from 'react';

interface Props {
  classes?: string;
  level?: 1 | 2 | 3 | 4 | 5;
  type: 'header' | 'paragraph';
}

const Typography: React.FC<React.PropsWithChildren<Props>> = ({
  classes,
  children,
  level,
  type,
}) => {
  let element = '';

  if (type === 'header') {
    if (level !== undefined) {
      element = `h${level}`;
    } else {
      element = 'h1';
    }
  } else {
    element = 'p';
  }

  return React.createElement(element, { className: classes }, children);
};

export default Typography;
