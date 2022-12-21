import { Card } from 'antd';
import React, { CSSProperties, ReactNode } from 'react';

interface CardProps {
  bodyStyle?: CSSProperties;
  children?: ReactNode;
  className?: string;
  extra?: ReactNode;
  headStyle?: CSSProperties;
  style?: CSSProperties;
  title?: string;
}

const CardComponent: React.FC<CardProps> = (props: CardProps) => {
  return (
    <Card {...props} />
  );
};

export default CardComponent;
