import React, {useMemo} from 'react';
import { useHistory, useParams } from 'react-router';
import { useStore } from 'contexts/Store';
import {RenderAllocationBarResourcePool} from 'components/ResourcePoolCardLight'

interface Params {
    poolname?: string;
}

const ResourcepoolDetail = () => {

    const { poolname } = useParams<Params>();
    const { agents, resourcePools } = useStore();

    const resourcePool = useMemo(()=>{
        return resourcePools.find(pool=>pool.name === poolname)
    }, [poolname, resourcePools])

    if(!resourcePool) return <div />

    return <RenderAllocationBarResourcePool resourcePool={resourcePool} />

}

export default ResourcepoolDetail