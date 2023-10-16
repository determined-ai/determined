import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import RadioGroup, { RadioGroupOption } from './RadioGroup';

const user = userEvent.setup();

const setup = (options: RadioGroupOption[], iconOnly = false) => {
  const handleOnChange = vi.fn();
  const view = render(
    <RadioGroup iconOnly={iconOnly} options={options} onChange={handleOnChange} />,
  );
  return { handleOnChange, view };
};

describe('RadioGroup', () => {
  const firstOption = 'First Option';
  const secondOption = 'Second Option';
  const radioOptions: RadioGroupOption[] = [
    { icon: 'learning', id: '1st', label: firstOption },
    { icon: 'heat', id: '2nd', label: secondOption },
  ];

  it('displays two radio button options with labels', () => {
    setup(radioOptions);
    expect(screen.getByText(firstOption)).toBeInTheDocument();
    expect(screen.getByText(secondOption)).toBeInTheDocument();
  });

  it('displays two radio button options without labels (icon only)', () => {
    setup(radioOptions, true);
    expect(() => screen.getByText(firstOption)).toThrow();
    expect(() => screen.getByText(secondOption)).toThrow();
  });

  it('updates state when radio button labels are clicked', async () => {
    const { handleOnChange, view } = setup(radioOptions);
    await user.click(await view.findByText(firstOption));
    expect(handleOnChange).toHaveBeenCalledWith('1st');
    await user.click(await view.findByText(secondOption));
    expect(handleOnChange).toHaveBeenCalledWith('2nd');
  });

  it('updates state when icon-only radio buttons are clicked', async () => {
    const { handleOnChange } = setup(radioOptions, true);
    await user.click(document.querySelectorAll('.ant-radio-button')[0]);
    expect(handleOnChange).toHaveBeenCalledWith('1st');
    await user.click(document.querySelectorAll('.ant-radio-button')[1]);
    expect(handleOnChange).toHaveBeenCalledWith('2nd');
  });
});
