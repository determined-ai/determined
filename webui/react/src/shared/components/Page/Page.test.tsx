import { render, screen } from '@testing-library/react';

import Page, { Props } from './Page';

const HEADER = 'header of page';
const CHILDREN = 'children of page';

const setup = (props: Props) => {
  const { container } = render(<Page {...props} />);
  return container;
};

describe('page functions', () => {
  it('should display page with header component', () => {
    setup({ headerComponent: <>{HEADER}</> });
    expect(screen.getByText(HEADER)).toBeInTheDocument();
  });
  it('should display spinner when loading', () => {
    const container = setup({ loading: true });
    expect(container.getElementsByClassName('ant-spin ant-spin-spinning')).toHaveLength(1);
  });
  it('should display children', () => {
    setup({ children: CHILDREN });
    expect(screen.getByText(CHILDREN)).toBeInTheDocument();
  });
  it('should use correct class name', () => {
    const container = setup({ bodyNoPadding: true, stickyHeader: true });
    expect(container.getElementsByClassName('base bodyNoPadding stickyHeader')).toHaveLength(1);
  });
});
