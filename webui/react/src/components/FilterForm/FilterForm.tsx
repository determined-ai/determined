import { useObservable } from 'micro-observables';

import { FormClassStore } from './FilterFormStore';
import FilterGroup from './FilterGroup';

interface Props {
  formClassStore: FormClassStore;
}

const FilterForm = ({ formClassStore }: Props): JSX.Element => {
  const data = useObservable(formClassStore.formset);
  return (
    <div>
      <FilterGroup
        conjunction={data.filterSet.conjunction}
        formClassStore={formClassStore}
        group={data.filterSet}
        index={0}
        level={0}
        parentId={data.filterSet.id}
      />
    </div>
  );
};

export default FilterForm;
