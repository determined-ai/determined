import { render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';

import { generateAlphaNumeric } from 'utils/string';

import BadgeTag from './BadgeTag';

const LABEL = generateAlphaNumeric();
const CONTENT = generateAlphaNumeric();
const CONTENT_TOOLTIP = generateAlphaNumeric();

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  /** We need to mock Tooltip in order to set mouseEnterDelay to 0
   */
  const Tooltip = (props: unknown) => {
    return (
      <antd.Tooltip
        {...props}
        mouseEnterDelay={0}
      />
    );
  };

  return {
    __esModule: true,
    ...antd,
    Tooltip,
  };
});

const setup = ({ labelAfter = true }) => {
  const view = render(
    <BadgeTag
      label={labelAfter && LABEL}
      preLabel={!labelAfter && LABEL}
      tooltip={CONTENT_TOOLTIP}>
      {CONTENT}
    </BadgeTag>,
  );
  return { view };
};

describe('BadgeTag', () => {
  it('displays label and content', () => {
    const { view } = setup({});

    expect(view.getByText(LABEL)).toBeInTheDocument();
    expect(view.getByText(CONTENT)).toBeInTheDocument();
  });
  it('displays prelabel', () => {
    const { view } = setup({ labelAfter: false });

    expect(view.getByText(LABEL)).toBeInTheDocument();
  });
  it('label displays tooltip on hover', async () => {
    const { view } = setup({});

    userEvent.hover(view.getByText(LABEL));
    // await waitFor(() => {
    //   expect(view.getByRole('tooltip').textContent).toEqual(LABEL);
    // });
    //await view.findByRole('tooltip');
    expect((await view.findByRole('tooltip')).textContent).toEqual(LABEL);
  });

  it('content displays tooltip on hover', async () => {
    const { view } = setup({});

    userEvent.hover(view.getByText(CONTENT));
    await waitFor(() => {
      expect(view.getByRole('tooltip').textContent).toEqual(CONTENT_TOOLTIP);
    });
  });
});
