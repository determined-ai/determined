import React from 'react';
import styled from 'styled-components';
import { switchProp } from 'styled-tools';

interface Props {
  name?: string;
  size?: 'small' | 'medium' | 'large';
}

const defaultProps: Props = {
  name: 'star',
  size: 'medium',
};

const sizeStyles = {
  large: '2.4rem',
  medium: '2rem',
  small: '1.2rem',
};

const Base = styled.i`
  font-size: ${switchProp('size', sizeStyles)};
  user-select: none;
`;

const Icon: React.FC<Props> = ({ name, ...props }: Props) => {
  return <Base className={`icon-${name}`} {...props} />;
};

Icon.defaultProps = defaultProps;

export default Icon;
