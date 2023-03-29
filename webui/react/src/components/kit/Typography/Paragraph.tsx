import React from 'react';

const Paragraph: React.FC<React.PropsWithChildren> = ({ children }) => {
  return <p>{children}</p>;
};

export default Paragraph;
