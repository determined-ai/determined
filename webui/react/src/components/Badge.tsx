import React, { PropsWithChildren } from 'react';
import styled, { css } from 'styled-components';
import { switchProp, theme } from 'styled-tools';

import { getStateColor } from 'themes';
import { CommandState, RunState } from 'types';
import { stateToLabel } from 'utils/types';

export enum BadgeType {
  Default,
  State,
}

interface Props {
  state?: RunState | CommandState;
  type?: BadgeType;
}

const defaultProps = {
  state: RunState.Active,
  type: BadgeType.Default,
};

const Badge: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return (
    <Base {...props}>
      {props.type === BadgeType.State && props.state ?
        stateToLabel(props.state) : props.children}
    </Base>
  );
};

const cssDefault = css`
  background-color: ${theme('colors.monochrome.7')};
`;

const cssState = css<Props>`
  background-color: ${(props): string => getStateColor(props.state, props.theme)};
  text-transform: uppercase;
`;

const typeStyles = {
  [BadgeType.Default]: cssDefault,
  [BadgeType.State]: cssState,
};

const Base = styled.span`
  border-radius: 3px;
  color: ${theme('colors.monochrome.17')};
  font-size: ${theme('sizes.font.tiny')};
  font-weight: bold;
  line-height: ${theme('sizes.font.large')};
  padding: 0 ${theme('sizes.layout.small')};
  text-align: center;
  ${switchProp('type', typeStyles)}
`;

Badge.defaultProps = defaultProps;

export default Badge;
