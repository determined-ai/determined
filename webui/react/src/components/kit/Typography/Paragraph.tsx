import React from 'react';

interface Props {
  className?: string;
  style?: { [k: string]: string };
}

const Paragraph: React.FC<React.PropsWithChildren<Props>> = ({ children, className, style }) => {
  return (
    <p className={className} style={style}>
      {children}
    </p>
  );
};

export default Paragraph;
