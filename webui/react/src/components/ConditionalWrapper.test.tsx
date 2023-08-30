import { render, screen } from '@testing-library/react';
import React, { ReactElement } from 'react';

import { ConditionalWrapper } from 'components/ConditionalWrapper';

const WRAPPER_ID = 'wrapper';
const FALSE_WRAPPER_ID = 'false-wrapper';
const CONTENT = <div>To wrap or not to wrap...</div>;

const wrapper = (children: ReactElement) => <div data-testid={WRAPPER_ID}>{children}</div>;

const falseWrapper = (children: ReactElement) => (
  <div data-testid={FALSE_WRAPPER_ID}>{children}</div>
);

describe('ConditionalWrapper', () => {
  it('renders true condition with wrapper', () => {
    render(
      <ConditionalWrapper condition={true} wrapper={wrapper}>
        {CONTENT}
      </ConditionalWrapper>,
    );
    expect(screen.queryByTestId(WRAPPER_ID)).toBeInTheDocument();
  });

  it('renders false condition without wrapper', () => {
    render(
      <ConditionalWrapper condition={false} wrapper={wrapper}>
        {CONTENT}
      </ConditionalWrapper>,
    );
    expect(screen.queryByTestId(WRAPPER_ID)).not.toBeInTheDocument();
  });

  it('renders false condition with alternative wrapper', () => {
    render(
      <ConditionalWrapper condition={false} falseWrapper={falseWrapper} wrapper={wrapper}>
        {CONTENT}
      </ConditionalWrapper>,
    );
    expect(screen.queryByTestId(WRAPPER_ID)).not.toBeInTheDocument();
    expect(screen.queryByTestId(FALSE_WRAPPER_ID)).toBeInTheDocument();
  });
});
