import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Spinner from 'hew/Spinner';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';

import { FilterFormStore } from './FilterFormStore';
import FilterGroup from './FilterGroup';

const filterFormStore = new FilterFormStore();

const Component = (): JSX.Element => {
  const loadableFormData = useObservable(filterFormStore.formset);

  return (
    <>
      {Loadable.match(loadableFormData, {
        Failed: () => null,
        Loaded: (data) => (
          <>
            <FilterGroup
              columns={[]}
              conjunction={data.filterGroup.conjunction}
              formStore={filterFormStore}
              group={data.filterGroup}
              index={0}
              level={0}
              parentId={data.filterGroup.id}
            />
          </>
        ),
        NotLoaded: () => <Spinner spinning />,
      })}
    </>
  );
};

const setup = () => {
  const user = userEvent.setup();

  render(
    <UIProvider theme={DefaultTheme.Light}>
      <Component />
    </UIProvider>,
  );

  return { user };
};

describe('FilterGroup', () => {
  it('should display spinner', async () => {
    setup();
    expect(await screen.findByTestId('custom-spinner')).toBeInTheDocument();
  });
});
