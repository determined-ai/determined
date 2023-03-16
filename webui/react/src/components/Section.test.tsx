import { render, screen } from '@testing-library/react';

import Section from './Section';

const setup = ({
  title = '',
  loading = false,
  hideTitle = false,
  options = <div />,
  filters = <div />,
  maxHeight = false,
  bodyBorder = false,
  bodyNoPadding = false,
  divider = false,
}) => {
  const handleOnChange = vi.fn();
  const view = render(
    <Section
      bodyBorder={bodyBorder}
      bodyNoPadding={bodyNoPadding}
      divider={divider}
      filters={filters}
      hideTitle={hideTitle}
      loading={loading}
      maxHeight={maxHeight}
      options={options}
      title={title}
    />,
  );
  return { handleOnChange, view };
};

describe('Section', () => {
  it('Section with title', () => {
    setup({ title: 'title of section' });
    expect(screen.getByText('title of section')).toBeInTheDocument();
  });

  it('Section hide title', () => {
    setup({ hideTitle: true, title: 'title of section' });
    expect(screen.queryAllByText('title of section')).toHaveLength(0);
  });

  it('Section with options', () => {
    setup({ options: <div data-testid="section-option" /> });
    expect(screen.getByTestId('section-option')).toBeInTheDocument();
  });

  it('Section with filters', () => {
    setup({ filters: <div data-testid="section-filters" /> });
    expect(screen.getByTestId('section-filters')).toBeInTheDocument();
  });

  it('Section in loading state', () => {
    const {
      view: { container },
    } = setup({
      filters: <div data-testid="section-filters" />,
      loading: true,
      title: 'section-title',
    });
    // Test that antd spinner is spinning
    expect(container.getElementsByClassName('ant-spin ant-spin-spinning')).toHaveLength(1);
    // Test that filter is not showing
    expect(screen.queryAllByTestId('section-filters')[0]).toBeEnabled();
  });

  it('Section with different styles', () => {
    setup({ bodyBorder: true, divider: true, maxHeight: true, title: 'section-title' });
    const section = screen.getByText('section-title') as HTMLElement;
    expect(section).toHaveStyle({ height: 100 });
    expect(section).toHaveStyle({
      border: 'solid var(--theme-stroke-width) var(--theme-colors-monochrome-12)',
    });
    expect(section).toHaveStyle({ borderTopWidth: 'var(--theme-stroke-width)' });
  });
});
