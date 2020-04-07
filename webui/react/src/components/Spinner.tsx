import React from 'react';
import styled, { keyframes } from 'styled-components';

import Icon from 'components/Icon';

interface Props {
  fullPage?: boolean;
}

const defaultProps: Props = { fullPage: false };

const Spinner: React.FC<Props> = (props: Props) => {
  return (
    <Base {...props}>
      <Spin>
        <Icon name="spinner" size="large" />
      </Spin>
    </Base>
  );
};

const rotate = keyframes`
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
`;

const Spin = styled.div`
  animation: ${rotate} 1s linear infinite;
  left: 50%;
  position: absolute;
  top: 50%;
  transform: translate(-50%, -50%);
`;

const Base = styled.div<Props>`
  background-color: ${(props): string => props.fullPage ? 'rgba(255, 255, 255, 0.8)' : ''};
  height: ${(props): string => props.fullPage ? '100%' : 'auto'};
  left: ${(props): string => props.fullPage ? '0' : 'auto'};
  position: ${(props): string => props.fullPage ? 'absolute' : 'relative'};
  top: ${(props): string => props.fullPage ? '0' : 'auto'};
  width: ${(props): string => props.fullPage ? '100%' : 'auto'};
`;

Spinner.defaultProps = defaultProps;

export default Spinner;
