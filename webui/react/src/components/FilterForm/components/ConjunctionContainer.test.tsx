import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SelectValue } from 'hew/Select';
import UIProvider, { DefaultTheme } from 'hew/Theme';

import ConjunctionContainer from 'components/FilterForm/components/ConjunctionContainer';

import { Conjunction } from './type';

const setup = ({
  index = 0,
  conjunction,
  onClick = vi.fn(),
}: {
  index?: number;
  conjunction: Conjunction;
  onClick?: (value: SelectValue) => void;
}) => {
  const user = userEvent.setup();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ConjunctionContainer conjunction={conjunction} index={index} onClick={onClick} />
    </UIProvider>,
  );

  return { user };
};

describe('ConjunctionContainer', () => {
  it('should show Where text when index is 0', () => {
    setup({ conjunction: Conjunction.Or });

    expect(screen.getByText('Where')).toHaveAttribute('data-test', 'where');
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  });

  it('should show Select when index is 1', async () => {
    const onClick = vi.fn();
    const { user } = setup({ conjunction: Conjunction.Or, index: 1, onClick });

    const conjunctionSelect = screen.getByRole('combobox');
    expect(screen.queryByText('Where')).not.toBeInTheDocument();
    expect(conjunctionSelect).toBeInTheDocument();
    expect(screen.getByText(Conjunction.Or)).toBeInTheDocument();
    expect(screen.queryByText(Conjunction.And)).not.toBeInTheDocument();

    expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
    await user.click(conjunctionSelect);
    expect(await screen.findByRole('listbox')).toBeInTheDocument();
    expect((await screen.findByRole('listbox')).children[0]).toHaveTextContent(Conjunction.And);
    await user.click((await screen.findAllByText(Conjunction.And))[1]);
    expect(onClick).toBeCalledWith(Conjunction.And, {
      children: Conjunction.And,
      key: Conjunction.And,
      value: Conjunction.And,
    });
  });

  it('should show Conjunction text when index is more than 1', () => {
    setup({ conjunction: Conjunction.Or, index: 2 });

    expect(screen.queryByText('Where')).not.toBeInTheDocument();
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.getByText(Conjunction.Or)).toHaveAttribute('data-test', 'conjunctionContinued');
  });
});
