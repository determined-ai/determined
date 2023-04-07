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
        conjunction={data.filterGroup.conjunction}
        formClassStore={formClassStore}
        group={data.filterGroup}
        index={0}
        level={0}
        parentId={data.filterGroup.id}
      />
      <div>{formClassStore.query}</div>
    </div>
  );
};

export default FilterForm;
