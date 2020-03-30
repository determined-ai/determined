import React, { PropsWithChildren } from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

interface Props {
  title: string;
}

const OverviewStats: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  return (
    <Base>
      <Title>{props.title}</Title>
      <Info>{props.children}</Info>
    </Base>
  );
};

const Title = styled.div`
  color: ${theme('colors.monochrome.7')};
  font-size: ${theme('sizes.font.tiny')};
  word-break: break-word;
`;

const Info = styled.div`
  color: ${theme('colors.monochrome.0')};
  font-size: ${theme('sizes.font.large')};
  line-height: 1;
  padding-top: ${theme('sizes.layout.medium')};
  & > small { font-size: ${theme('sizes.font.medium')}; }
`;

const Base = styled.div`
  background-color: ${theme('colors.monochrome.13')};
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: ${theme('sizes.layout.medium')};
`;

export default OverviewStats;
