import React, { PropsWithChildren } from 'react';
import styled, { css } from 'styled-components';
import { ifProp, theme } from 'styled-tools';

import LayoutHelper from 'components/LayoutHelper';
import { toHtmlId } from 'utils/string';

interface Props {
  divider?: boolean;
  options?: React.ReactNode;
  title: string;
}

const defaultProps = {
  divider: false,
};

const Section: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return (
    <Base data-test="section" id={toHtmlId(props.title)}>
      <Header data-test="header">
        <Title>{props.title}</Title>
        {props.options && <LayoutHelper>{props.options}</LayoutHelper>}
      </Header>
      <Body data-test="body" divider={props.divider || false}>
        {props.children}
      </Body>
    </Base>
  );
};

const Base = styled.section`
  padding-bottom: ${theme('sizes.layout.giant')};
`;

const Title = styled.h2`
  font-size: ${theme('sizes.layout.jumbo')};
  font-weight: bold;
  margin: 0;
`;

const Header = styled.div`
  align-items: center;
  display: flex;
  justify-content: space-between;
  padding-bottom: ${theme('sizes.layout.medium')};
`;

const dividerPadding = css`
  padding: ${theme('sizes.layout.big')} 0;
`;

const Body = styled.div<{ divider: boolean }>`
  border-color: ${theme('colors.monochrome.13')};
  border-style: solid;
  border-width: 0;
  border-top-width: ${ifProp('divider', theme('sizes.border.width'), '0')};
  ${ifProp('divider', dividerPadding, '')}
`;

Section.defaultProps = defaultProps;

export default Section;
