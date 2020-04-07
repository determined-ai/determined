import React from 'react';
import styled from 'styled-components';
import { prop } from 'styled-tools';

import { getStateColor } from 'themes';
import { CommandState, RunState } from 'types';

interface Props {
  percent: number;
  state: RunState | CommandState;
}

const defaultProps = {
  percent: 0,
};

const ProgressBar: React.FC<Props> = (props: Props) => {
  return (
    <Base {...props}>
      <span />
    </Base>
  );
};

const Base = styled.div<Props>`
  background: #ddd;
  border-radius: 0;
  box-shadow: inset 0 -1px 1px rgba(255, 255, 255, 0.3);
  height: 4px;
  margin: 0;
  padding: 0;
  position: relative;
  & > span {
    background-color: ${(props): string => getStateColor(props.state, props.theme)};
    box-shadow:
      inset 0 2px 9px  rgba(255, 255, 255, 0.3),
      inset 0 -2px 6px rgba(0, 0, 0, 0.4);
    display: block;
    height: 100%;
    overflow: hidden;
    position: relative;
    width: ${prop('percent', defaultProps.percent)}%;
  }
`;

ProgressBar.defaultProps = defaultProps;

export default ProgressBar;
