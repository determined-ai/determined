import { boolean, withKnobs } from '@storybook/addon-knobs';
import React from 'react';
import styled from 'styled-components';

import Icon from 'components/Icon';
import { ShirtSize } from 'themes';

import LayoutHelper from './LayoutHelper';

export default {
  component: LayoutHelper,
  decorators: [ withKnobs ],
  title: 'LayoutHelper',
};

const Page = styled.div`
  background-color: #eee;
  border: 1px #666 solid;
  height: 30rem;
  text-align: center;
  width: 30rem;
`;

export const Default = (): React.ReactNode => (
  <Page>
    <LayoutHelper>
      <Icon name="experiment" />
    </LayoutHelper>
  </Page>
);

export const DefaultLargePadding = (): React.ReactNode => (
  <Page>
    <LayoutHelper padding={[ ShirtSize.large ]}>
      <Icon name="experiment" />
    </LayoutHelper>
  </Page>
);

export const DefaultMultipleChildren= (): React.ReactNode => (
  <Page>
    <LayoutHelper>
      <Icon name="experiment" />
      <Icon name="tensorboard" />
      <Icon name="notebook" />
      <Icon name="cluster" />
    </LayoutHelper>
  </Page>
);

export const MultipleChildrenWithKnobs = (): React.ReactNode => (
  <Page>
    <LayoutHelper
      center={boolean('center', false)}
      column={boolean('column', false)}
      fullHeight={boolean('fullHeight', false)}
      grow={boolean('grow', false)}
      xCenter={boolean('xCenter', false)}
      xEnd={boolean('xEnd', false)}
      xStart={boolean('xStart', false)}
      yCenter={boolean('yCenter', false)}
      yEnd={boolean('yEnd', false)}
      yStart={boolean('yStart', false)}
    >
      <Icon name="experiment" />
      <Icon name="tensorboard" />
      <Icon name="notebook" />
      <Icon name="cluster" />
    </LayoutHelper>
  </Page>
);

export const MultipleChildrenXCenterGrow = (): React.ReactNode => (
  <Page>
    <LayoutHelper grow xCenter>
      <Icon name="experiment" />
      <Icon name="tensorboard" />
      <Icon name="notebook" />
      <Icon name="cluster" />
    </LayoutHelper>
  </Page>
);

export const MultipleChildrenXCenterVertical= (): React.ReactNode => (
  <Page>
    <LayoutHelper column xCenter>
      <Icon name="experiment" />
      <Icon name="tensorboard" />
      <Icon name="notebook" />
      <Icon name="cluster" />
    </LayoutHelper>
  </Page>
);

export const xyCenter = (): React.ReactNode => (
  <Page>
    <LayoutHelper center fullHeight>
      <Icon name="experiment" />
      <Icon name="tensorboard" />
      <Icon name="notebook" />
      <Icon name="cluster" />
    </LayoutHelper>
  </Page>
);
