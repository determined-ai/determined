import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Spinner from 'hew/Spinner';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';

import { FilterFormStore, ROOT_ID } from './FilterFormStore';
import FilterGroup from './FilterGroup';
import { FormKind } from './type';

const filterFormStore = new FilterFormStore();

const Component = ({ filterFormStore }: { filterFormStore: FilterFormStore }): JSX.Element => {
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
      <DndProvider backend={HTML5Backend}>
        <Component filterFormStore={filterFormStore} />
      </DndProvider>
    </UIProvider>,
  );

  return { user };
};

describe('FilterGroup', () => {
  describe('before init', () => {
    it('should display spinner', async () => {
      setup();
      expect(await screen.findByTestId('custom-spinner')).toBeInTheDocument();
    });
  });

  describe('after init', () => {
    beforeEach(() => {
      filterFormStore.init();
    });

    it('should display group', async () => {
      setup();
      filterFormStore.addChild(ROOT_ID, FormKind.Group);
      expect(screen.queryByTestId('custom-spinner')).not.toBeInTheDocument();
      expect(await screen.findByText('All of the following are true...')).toBeInTheDocument();
    });

    it('should not display group when field is added', () => {
      setup();
      filterFormStore.addChild(ROOT_ID, FormKind.Field);
      expect(screen.queryByTestId('custom-spinner')).not.toBeInTheDocument();
      expect(screen.queryByText('All of the following are true...')).not.toBeInTheDocument();
    });

    it('should add a field in a group', async () => {
      const { user } = setup();
      filterFormStore.addChild(ROOT_ID, FormKind.Group);
      await user.click((await screen.findAllByRole('button'))[0]);
      await user.click(await screen.findByText('Add condition'));
      expect((await screen.findAllByText('Where')).length).toBe(2);
    });

    it('should add a group in a group', async () => {
      const { user } = setup();
      filterFormStore.addChild(ROOT_ID, FormKind.Group);
      await user.click((await screen.findAllByRole('button'))[0]);
      await user.click(await screen.findByText('Add condition group'));
      expect((await screen.findAllByText('All of the following are true...')).length).toBe(2);
    });
  });
});
