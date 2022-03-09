import { fireEvent, render, screen } from '@testing-library/react';
import React from 'react';

import { RadioGroupOption } from 'types';

import RadioGroup from './RadioGroup';

const setup = (options: RadioGroupOption[], iconOnly = false) => {
  const handleOnChange = jest.fn();
  const view = render(
    <RadioGroup
      iconOnly={iconOnly}
      options={options}
      onChange={handleOnChange}
    />,
  );
  return { handleOnChange, view };
};

describe('RadioGroup', () => {
  const radioOptions: RadioGroupOption[] = [
    { icon: 'learning', id: '1st', label: 'First Option' },
    { icon: 'heat', id: '2nd', label: 'Second Option' },
  ];

  it('displays two radio button options with labels', () => {
    setup(radioOptions);
    expect(screen.getByText('First Option')).toBeInTheDocument();
    expect(screen.getByText('Second Option')).toBeInTheDocument();
  });

  it('displays two radio button options without labels (icon only)', () => {
    setup(radioOptions, true);
    expect(() => screen.getByText('First Option')).toThrow();
    expect(() => screen.getByText('Second Option')).toThrow();
  });

  it('updates state when radio button labels are clicked', async () => {
    const { handleOnChange, view } = setup(radioOptions);
    fireEvent.click(await view.findByText('First Option'));
    expect(handleOnChange.mock.calls).toHaveLength(1);
    expect(handleOnChange.mock.calls[0][0]).toBe('1st');
    fireEvent.click(await view.findByText('Second Option'));
    expect(handleOnChange.mock.calls).toHaveLength(2);
    expect(handleOnChange.mock.calls[1][0]).toBe('2nd');
  });

  it('updates state when icon-only radio buttons are clicked', () => {
    const { handleOnChange } = setup(radioOptions, true);
    fireEvent.click(document.querySelectorAll('.ant-radio-button')[0]);
    expect(handleOnChange.mock.calls).toHaveLength(1);
    expect(handleOnChange.mock.calls[0][0]).toBe('1st');
    fireEvent.click(document.querySelectorAll('.ant-radio-button')[1]);
    expect(handleOnChange.mock.calls).toHaveLength(2);
    expect(handleOnChange.mock.calls[1][0]).toBe('2nd');
  });
});
