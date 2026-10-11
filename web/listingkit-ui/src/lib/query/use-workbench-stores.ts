"use client";

import {
  useMutation,
  useQuery,
  useQueryClient,
  type MutateOptions,
  type UseMutationResult,
} from "@tanstack/react-query";
import { useRef, useState } from "react";
import { z } from "zod";
import { workbenchStoreCreateSchema } from "@/lib/validation/workbench-store";

import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import {
  WorkbenchAPIError,
  createWorkbenchStore,
  deleteWorkbenchStore,
  disableWorkbenchStore,
  enableWorkbenchStore,
  getWorkbenchStore,
  listWorkbenchStores,
  updateWorkbenchStore,
  type WorkbenchStoreCreateInput,
  type WorkbenchStore,
  type WorkbenchStoreDeleteResult,
  type WorkbenchStoreListFilters,
  type WorkbenchStoreUpdateInput,
} from "@/lib/api/workbench-stores";

export const workbenchStoreKeys = {
  root: (organizationId: string) =>
    ["workbench", organizationId, "stores"] as const,
  list: (organizationId: string, filters: WorkbenchStoreListFilters) =>
    [...workbenchStoreKeys.root(organizationId), "list", filters] as const,
  item: (organizationId: string, storeId: string) =>
    [...workbenchStoreKeys.root(organizationId), "item", storeId] as const,
  mutation: (organizationId: string, action: string) =>
    [...workbenchStoreKeys.root(organizationId), "mutation", action] as const,
};

type CapturedOperation<T> = {
  organizationId: string;
  input: T;
};

type CapturedKeyedOperation<T> = CapturedOperation<T> & {
  operationKey: string;
  persistenceKey?: string;
};

type StoreVersionInput = { id: string; version: number };
type StoreUpdateInput = StoreVersionInput & {
  input: WorkbenchStoreUpdateInput;
};

export function useWorkbenchStores(filters: WorkbenchStoreListFilters) {
  const organizationId = useEffectiveOrganizationId();
  const { permissions } = useWorkbenchContext();
  return useQuery({
    queryKey: [...workbenchStoreKeys.list(organizationId, filters), permissions],
    queryFn: () => listWorkbenchStores(filters, organizationId),
    enabled: organizationId.length > 0,
    retry: retryWorkbenchRequest,
  });
}

export function useWorkbenchStore(storeId: string) {
  const organizationId = useEffectiveOrganizationId();
  const { permissions } = useWorkbenchContext();
  return useQuery({
    queryKey: [...workbenchStoreKeys.item(organizationId, storeId), permissions],
    queryFn: () => getWorkbenchStore(storeId, organizationId),
    enabled: organizationId.length > 0 && storeId.length > 0,
    retry: retryWorkbenchRequest,
  });
}

const savedCreateSchema=z.object({organizationId:z.string().min(1).max(128),operationKey:z.string().uuid(),input:workbenchStoreCreateSchema,persistenceKey:z.string().max(500)}).strict();
function readSavedCreate(key:string){
 const raw=sessionStorage.getItem(key);if(!raw)return null;if(raw.length>4096)throw Error("invalid saved intent");const operation=savedCreateSchema.parse(JSON.parse(raw));if(operation.persistenceKey!==key)throw Error("wrong saved scope");return operation;
}
export function useCreateWorkbenchStore({persist=false}:{persist?:boolean}={}) {
  const organizationId = useEffectiveOrganizationId();
  const userId=useWorkbenchContext().user?.id??"";
  const persistenceKey=persist?`store-create:${JSON.stringify([organizationId,userId])}`:"";
  const [restored]=useState(()=>{if(!persist)return {operation:null,error:false,key:""};try{if(!organizationId||!userId)throw Error("missing scope");return {operation:readSavedCreate(persistenceKey),error:false,key:persistenceKey}}catch{return {operation:null,error:true,key:persistenceKey}}});
  const storageError=persist&&(restored.error||restored.key!==persistenceKey);
  const queryClient = useQueryClient();
  const lastOperationRef = useRef<{
    operation: CapturedKeyedOperation<WorkbenchStoreCreateInput>;
    retryable: boolean;
  } | null>(restored.operation?{operation:restored.operation,retryable:true}:null);
  const [retryState, setRetryState] = useState<RetryState | null>(restored.operation?{organizationId,available:true}:null);
  const mutation = useMutation<
    WorkbenchStore,
    WorkbenchAPIError,
    CapturedKeyedOperation<WorkbenchStoreCreateInput>
  >({
    mutationKey: workbenchStoreKeys.mutation(organizationId, "create"),
    mutationFn: (operation: CapturedKeyedOperation<WorkbenchStoreCreateInput>) => {
      requireEffectiveOrganization(operation.organizationId);
      if(persist){
        if(storageError||!operation.persistenceKey)throw new WorkbenchAPIError(0,"STORE_CREATE_OUTCOME_UNKNOWN","无法安全保存原创建键，请刷新后恢复", "",[]);
        const saved=readSavedCreate(operation.persistenceKey);
        if(saved && saved.operationKey!==operation.operationKey)throw new WorkbenchAPIError(0,"STORE_CREATE_OUTCOME_UNKNOWN","原创建结果仍待确认", "",[]);
        sessionStorage.setItem(operation.persistenceKey,JSON.stringify(operation));
      }
      return createWorkbenchStore(operation.input, operation.operationKey, operation.organizationId);
    },
    retry: retryWorkbenchRequest,
    onSuccess: (_store, operation) =>
      {
        if(operation.persistenceKey && readSavedCreate(operation.persistenceKey)?.operationKey===operation.operationKey)sessionStorage.removeItem(operation.persistenceKey);
        updateLastKeyedOperation(lastOperationRef, setRetryState, operation, false);
        return invalidateCapturedOrganization(queryClient, operation.organizationId);
      },
    onError: (error, operation) => {
      if(operation.persistenceKey){
        if((error.code==="INVALID_REQUEST" && error.status===400) || (error.code==="STORE_ALREADY_EXISTS" && error.status===409)){
          if(readSavedCreate(operation.persistenceKey)?.operationKey===operation.operationKey)sessionStorage.removeItem(operation.persistenceKey);
        }
        else {let saved=operation;try{saved=readSavedCreate(operation.persistenceKey)??operation}catch{}if(lastOperationRef.current?.operation===operation){lastOperationRef.current={operation:saved,retryable:true};setRetryState({organizationId:saved.organizationId,available:true})}return}
      }
      updateLastKeyedOperation(
        lastOperationRef,
        setRetryState,
        operation,
        isExplicitRetryEligible(error),
      );
    },
  });
  const submit = (
    input: WorkbenchStoreCreateInput,
    options?: MutationOptions<WorkbenchStore, WorkbenchStoreCreateInput>,
  ) => {
    const operation = {...captureKeyedOperation(organizationId, input),...(persist?{persistenceKey}: {})};
    lastOperationRef.current = { operation, retryable: false };
    setRetryState({ organizationId: operation.organizationId, available: false });
    return [operation, captureMutationOptions(options, input)] as const;
  };
  const retryLast = () => retryLastKeyedOperation(
    mutation,
    lastOperationRef,
    setRetryState,
    organizationId,
  );
  return {
    ...mutation,
    variables: mutation.variables?.input,
    mutate: (
      input: WorkbenchStoreCreateInput,
      options?: MutationOptions<WorkbenchStore, WorkbenchStoreCreateInput>,
    ) =>
      mutation.mutate(...submit(input, options)),
    mutateAsync: (
      input: WorkbenchStoreCreateInput,
      options?: MutationOptions<WorkbenchStore, WorkbenchStoreCreateInput>,
    ) =>
      organizationId
        ? mutation.mutateAsync(...submit(input, options))
        : Promise.reject(missingOrganizationError()),
    retryLast,
    storageError,
    canRetryLast: !storageError && retryState?.available === true && retryState.organizationId === organizationId,
  };
}

export function useUpdateWorkbenchStore() {
  const organizationId = useEffectiveOrganizationId();
  const queryClient = useQueryClient();
  const mutation = useMutation<
    WorkbenchStore,
    WorkbenchAPIError,
    CapturedOperation<StoreUpdateInput>
  >({
    mutationKey: workbenchStoreKeys.mutation(organizationId, "update"),
    mutationFn: (operation: CapturedOperation<StoreUpdateInput>) => {
      requireEffectiveOrganization(operation.organizationId);
      const { id, input, version } = operation.input;
      return updateWorkbenchStore(id, input, version, operation.organizationId);
    },
    retry: retryWorkbenchRequest,
    onSuccess: (_store, operation) =>
      invalidateCapturedOrganization(queryClient, operation.organizationId),
  });
  return exposeMutation(mutation, (input: StoreUpdateInput) =>
    captureOperation(organizationId, input),
  );
}

export function useEnableWorkbenchStore() {
  return useStoreStateMutation(enableWorkbenchStore);
}

export function useDisableWorkbenchStore() {
  return useStoreStateMutation(disableWorkbenchStore);
}

export function useDeleteWorkbenchStore() {
  const organizationId = useEffectiveOrganizationId();
  const queryClient = useQueryClient();
  const lastOperationRef = useRef<{
    operation: CapturedKeyedOperation<StoreVersionInput>;
    retryable: boolean;
  } | null>(null);
  const [retryState, setRetryState] = useState<RetryState | null>(null);
  const mutation = useMutation<
    WorkbenchStoreDeleteResult,
    WorkbenchAPIError,
    CapturedKeyedOperation<StoreVersionInput>
  >({
    mutationKey: workbenchStoreKeys.mutation(organizationId, "delete"),
    mutationFn: (operation: CapturedKeyedOperation<StoreVersionInput>) => {
      requireEffectiveOrganization(operation.organizationId);
      return deleteWorkbenchStore(
        operation.input.id,
        operation.input.version,
        operation.operationKey,
        operation.organizationId,
      );
    },
    retry: retryWorkbenchRequest,
    onSuccess: (_result: WorkbenchStoreDeleteResult, operation) =>
      {
        updateLastKeyedOperation(lastOperationRef, setRetryState, operation, false);
        return invalidateCapturedOrganization(queryClient, operation.organizationId);
      },
    onError: (error, operation) => {
      updateLastKeyedOperation(
        lastOperationRef,
        setRetryState,
        operation,
        isExplicitRetryEligible(error),
      );
    },
  });
  const submit = (
    input: StoreVersionInput,
    options?: MutationOptions<WorkbenchStoreDeleteResult, StoreVersionInput>,
  ) => {
    const operation = captureKeyedOperation(organizationId, input);
    lastOperationRef.current = { operation, retryable: false };
    setRetryState({ organizationId: operation.organizationId, available: false });
    return [operation, captureMutationOptions(options, input)] as const;
  };
  const retryLast = () => retryLastKeyedOperation(
    mutation,
    lastOperationRef,
    setRetryState,
    organizationId,
  );
  const resume = (input: StoreVersionInput) =>
    organizationId
      ? mutation.mutateAsync(...submit(input))
      : Promise.reject(missingOrganizationError());
  return {
    ...mutation,
    variables: mutation.variables?.input,
    mutate: (
      input: StoreVersionInput,
      options?: MutationOptions<WorkbenchStoreDeleteResult, StoreVersionInput>,
    ) =>
      mutation.mutate(...submit(input, options)),
    mutateAsync: (
      input: StoreVersionInput,
      options?: MutationOptions<WorkbenchStoreDeleteResult, StoreVersionInput>,
    ) =>
      organizationId
        ? mutation.mutateAsync(...submit(input, options))
        : Promise.reject(missingOrganizationError()),
    retryLast,
    resume,
    canRetryLast: retryState?.available === true && retryState.organizationId === organizationId,
  };
}

function useStoreStateMutation(
  action: (storeId: string, version: number, expectedOrganizationId: string) => ReturnType<
    typeof enableWorkbenchStore
  >,
  actionName = action === enableWorkbenchStore ? "enable" : "disable",
) {
  const organizationId = useEffectiveOrganizationId();
  const queryClient = useQueryClient();
  const mutation = useMutation<
    WorkbenchStore,
    WorkbenchAPIError,
    CapturedOperation<StoreVersionInput>
  >({
    mutationKey: workbenchStoreKeys.mutation(
      organizationId,
      actionName,
    ),
    mutationFn: (operation: CapturedOperation<StoreVersionInput>) => {
      requireEffectiveOrganization(operation.organizationId);
      return action(operation.input.id, operation.input.version, operation.organizationId);
    },
    retry: retryWorkbenchRequest,
    onSuccess: (_store, operation) =>
      invalidateCapturedOrganization(queryClient, operation.organizationId),
  });
  return exposeMutation(mutation, (input: StoreVersionInput) =>
    captureOperation(organizationId, input),
  );
}

type MutationOptions<TData, TInput> = MutateOptions<
  TData,
  WorkbenchAPIError,
  TInput,
  unknown
>;

function exposeMutation<
  TData,
  TInput,
  TOperation extends CapturedOperation<TInput>,
>(
  mutation: UseMutationResult<TData, WorkbenchAPIError, TOperation, unknown>,
  capture: (input: TInput) => TOperation,
) {
  return {
    ...mutation,
    variables: mutation.variables?.input,
    mutate: (input: TInput, options?: MutationOptions<TData, TInput>) =>
      mutation.mutate(capture(input), captureMutationOptions(options, input)),
    mutateAsync: (input: TInput, options?: MutationOptions<TData, TInput>) =>
      mutation.mutateAsync(
        capture(input),
        captureMutationOptions(options, input),
      ),
  };
}

function captureKeyedOperation<T>(
  organizationId: string,
  input: T,
): CapturedKeyedOperation<T> {
  return {
    organizationId,
    input,
    operationKey: organizationId ? globalThis.crypto.randomUUID() : "",
  };
}

function isExplicitRetryEligible(error: WorkbenchAPIError) {
  return error.status === 0 || error.status >= 500;
}

function canRetryLastKeyedOperation<T>(
  last: { operation: CapturedKeyedOperation<T>; retryable: boolean } | null,
  organizationId: string,
) {
  return Boolean(last?.retryable && last.operation.organizationId === organizationId);
}

function updateLastKeyedOperation<T>(
  ref: React.MutableRefObject<{
    operation: CapturedKeyedOperation<T>;
    retryable: boolean;
  } | null>,
  setRetryState: React.Dispatch<React.SetStateAction<RetryState | null>>,
  operation: CapturedKeyedOperation<T>,
  retryable: boolean,
) {
  if (ref.current?.operation !== operation) return;
  ref.current = { operation, retryable };
  setRetryState({ organizationId: operation.organizationId, available: retryable });
}

function retryLastKeyedOperation<TData, TInput>(
  mutation: UseMutationResult<
    TData,
    WorkbenchAPIError,
    CapturedKeyedOperation<TInput>,
    unknown
  >,
  ref: React.MutableRefObject<{
    operation: CapturedKeyedOperation<TInput>;
    retryable: boolean;
  } | null>,
  setRetryState: React.Dispatch<React.SetStateAction<RetryState | null>>,
  organizationId: string,
) {
  const last = ref.current;
  if (!last || !canRetryLastKeyedOperation(last, organizationId)) {
    return Promise.reject(
      new WorkbenchAPIError(0, "INVALID_REQUEST", "No Store retry is available", "", []),
    );
  }
  ref.current = { ...last, retryable: false };
  setRetryState({ organizationId: last.operation.organizationId, available: false });
  return mutation.mutateAsync(last.operation);
}

type RetryState = { organizationId: string; available: boolean };

function captureOperation<T>(organizationId: string, input: T): CapturedOperation<T> {
  return { organizationId, input };
}

function requireEffectiveOrganization(organizationId: string) {
  if (organizationId) return;
  throw missingOrganizationError();
}

function missingOrganizationError() {
  return new WorkbenchAPIError(
    409,
    "ORGANIZATION_SELECTION_REQUIRED",
    "An Organization selection is required",
    "",
    [],
  );
}

function captureMutationOptions<TData, TInput, TOperation>(
  options: MutationOptions<TData, TInput> | undefined,
  input: TInput,
): MutateOptions<TData, WorkbenchAPIError, TOperation, unknown> | undefined {
  if (!options) return undefined;
  return {
    onSuccess: options.onSuccess
      ? (data, _operation, onMutateResult, context) =>
          options.onSuccess?.(data, input, onMutateResult, context)
      : undefined,
    onError: options.onError
      ? (error, _operation, onMutateResult, context) =>
          options.onError?.(error, input, onMutateResult, context)
      : undefined,
    onSettled: options.onSettled
      ? (data, error, _operation, onMutateResult, context) =>
          options.onSettled?.(data, error, input, onMutateResult, context)
      : undefined,
  };
}

function useEffectiveOrganizationId() {
  return useWorkbenchContext().effectiveOrganization?.id ?? "";
}

function retryWorkbenchRequest(failureCount: number, error: unknown) {
  return (
    failureCount < 2 &&
    error instanceof WorkbenchAPIError &&
    (error.status === 0 || error.status >= 500)
  );
}

function invalidateCapturedOrganization(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: string,
) {
  return queryClient.invalidateQueries({
    queryKey: workbenchStoreKeys.root(organizationId),
  });
}
