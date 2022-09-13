import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import StoreProvider from 'contexts/Store';
import { generateAlphaNumeric } from 'shared/utils/string';
import { StateOfUnion } from 'themes';

import { BadgeType } from './Badge';
import BadgeTag, { Props } from './BadgeTag';

const LABEL = generateAlphaNumeric();
const CONTENT = generateAlphaNumeric();
const CONTENT_TOOLTIP = generateAlphaNumeric();

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  // We need to mock Tooltip in order to set mouseEnterDelay to 0.
  const Tooltip = (props: {
    label?: React.ReactNode;
    preLabel?: React.ReactNode;
    state?: StateOfUnion | undefined;
    type?: BadgeType | undefined;
  }) => {
    return (
      <antd.Tooltip {...props} mouseEnterDelay={0} />
    );
  };

  return {
    __esModule: true,
    ...antd,
    Tooltip,
  };
});

const setup = ({
  children = CONTENT,
  tooltip = CONTENT_TOOLTIP,
  ...props
}: Props = {}) => {
  const view = render(
    <StoreProvider>
      <BadgeTag tooltip={tooltip} {...props}>{children}</BadgeTag>
    </StoreProvider>,
  );
  return { view };
};

describe('BadgeTag', () => {
  it('displays label and content', () => {
    const { view } = setup({ label: LABEL });
    expect(view.getByText(LABEL)).toBeInTheDocument();
    expect(view.getByText(CONTENT)).toBeInTheDocument();
  });

  it('displays prelabel', () => {
    const { view } = setup({ preLabel: LABEL });
    expect(view.getByText(LABEL)).toBeInTheDocument();
  });

  it('label displays tooltip on hover', async () => {
    const { view } = setup({ label: LABEL });
    userEvent.hover(view.getByText(LABEL));
    expect((await view.findByRole('tooltip')).textContent).toEqual(LABEL);
  });

  it('content displays tooltip on hover', async () => {
    const { view } = setup({ label: LABEL });
    userEvent.hover(view.getByText(CONTENT));
    expect((await view.getByRole('tooltip')).textContent).toEqual(CONTENT_TOOLTIP);
  });
});
