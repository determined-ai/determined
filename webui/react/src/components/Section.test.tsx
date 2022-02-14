import { render, screen } from '@testing-library/react';
import React from 'react';

import Section from './Section';

const setup = ({ 
    title='', 
    loading=false,
    hideTitle=false, 
    options=<div />, 
    filters=<div />, 
    maxHeight=false, 
    bodyBorder=false, 
    bodyNoPadding=false,
    divider=false,
}) => {
  const handleOnChange = jest.fn();
  const view = render(
    <Section 
        loading={loading}
        title={title} 
        hideTitle={hideTitle} 
        options={options} 
        filters={filters} 
        maxHeight={maxHeight} 
        bodyBorder={bodyBorder}
        bodyNoPadding={bodyNoPadding}
        divider={divider} />,
  );
  return { handleOnChange, view };
};

describe('Section', () => {

  it('Section with title', () => {
    setup({ title: 'title of section' });
    expect(screen.getByText('title of section')).toBeInTheDocument();
  });

  it('Section hide title', () => {
    setup({ title: 'title of section', hideTitle: true });
    expect(screen.queryAllByText('title of section').length == 0);
  });

  it('Section in loading state', () => {
    setup({ loading: true, title: 'section-title' });
    expect(screen.getByText('section-title')).toBeInTheDocument();
  });

  it('Section with options', () => {
    setup({ options: <div data-testid='section-option' /> });
    expect(screen.getByTestId('section-option')).toBeInTheDocument();
  });

  it('Section with filters', () => {
    setup({ filters: <div data-testid='section-filters' /> });
    expect(screen.getByTestId('section-filters')).toBeInTheDocument();
  });

  it('Section with different styles', () => {
    setup({ title: 'section-title', maxHeight: true, bodyBorder: true, divider: true });
    const section = screen.getByText('section-title') as HTMLElement;
    expect(section).toHaveStyle({height: 100 });
    expect(section).toHaveStyle({border: 'solid var(--theme-sizes-border-width) var(--theme-colors-monochrome-12)' });
    expect(section).toHaveStyle({borderTopWidth: 'var(--theme-sizes-border-width)'})
    // expect(section).toHaveStyle({})
  });


});
