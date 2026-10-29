import {render,screen,cleanup} from '@testing-library/react';
import {afterEach,expect,it} from 'vitest';
import {SourceQuotes,VariantSourcePrice} from './source-quotes';

afterEach(cleanup);
it('shows both conditions and original used grade on saved quote facts',()=>{
 const price={amount:'35.98',currency:'USD',minQuantity:null,condition:'new' as const,alternative:{amount:'31.99',currency:null,condition:'used' as const,conditionNote:'Very Good'}};
 render(<><SourceQuotes showUnknown quotes={{state:'available',publicationId:'fixed',version:'1',prices:[price]}}/><VariantSourcePrice showUnknown attributes={[{name:'source.price.fact',value:JSON.stringify(price)}]}/></>);
 for(const row of screen.getAllByText(/35\.98/)){expect(row).toHaveTextContent('新品');expect(row).toHaveTextContent('31.99');expect(row).toHaveTextContent('二手');expect(row).toHaveTextContent('Very Good');}
});
it('marks historical Amazon facts as unknown without guessing',()=>{
 render(<VariantSourcePrice showUnknown attributes={[{name:'source.price.fact',value:'{"amount":"19.99","currency":"USD"}'}]}/>);
 expect(screen.getByText(/19\.99/)).toHaveTextContent('状态未记录');
});
