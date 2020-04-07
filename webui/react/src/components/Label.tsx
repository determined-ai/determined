import React, { PropsWithChildren } from 'react';
import styled, { css } from 'styled-components';
import { switchProp } from 'styled-tools';

export enum LabelTypes {
  NavMain = 'nav-main',
  NavSideBar = 'nav-side-bar',
}

interface Props {
  type?: LabelTypes;
}

const cssNavMain = css`
  cursor: pointer;
  font-size: 1.2rem;
  font-weight: bold;
`;

const cssNavSideBar = css`
  cursor: pointer;
`;

const typeStyles = {
  [LabelTypes.NavMain]: cssNavMain,
  [LabelTypes.NavSideBar]: cssNavSideBar,
};

const Base = styled.div`
  ${switchProp('type', typeStyles)}
`;

const Label: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return React.createElement(Base, props, props.children);
};

export default Label;
