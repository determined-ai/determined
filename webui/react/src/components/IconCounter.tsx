import React from 'react';
import styled, {  css } from 'styled-components';
import { switchProp, theme } from 'styled-tools';

import Icon from 'components/Icon';

// IconCounter component.
interface Props {
  name: string;
  count: number;
  type: IconCounterType;
  onClick: () => void;
}

export enum IconCounterType {
  Active = 'active',
  Disabled = 'disabled',
}

const IconCounter: React.FC<Props> = (props: Props) => {
  return (
    <Base {...props} onClick={props.onClick}>
      <Icon name={props.name} size="large" />
      <Count>{props.count}</Count>
    </Base>
  );
};

const cssDisabled = css`
  color: ${theme('colors.monochrome.9')};
  &:hover { color: ${theme('colors.monochrome.9')}; }
`;

const cssActive = css`
  color: ${theme('colors.active')};
`;

const typeStyles = {
  [IconCounterType.Active]: cssActive,
  [IconCounterType.Disabled]: cssDisabled,
};

const Base = styled.a`
  align-items: center;
  cursor: pointer;
  display: grid;
  grid-gap: ${theme('sizes.layout.small')};
  grid-template-columns: 1fr auto;
  user-select: none;
  ${switchProp('type', typeStyles)}
`;

const Count = styled.span`
  font-size: ${theme('sizes.font.medium')};
  font-weight: bold;
`;

export default IconCounter;
