### featch path:

Formal ← user_input
stage 00 -> BRD (Business Requirements Document) | SysA
    ↓
Feasibility Study ← Current State Architecture (As-Is HLD)
stage 01 -> (ТЭО – Технико-экономическое обоснование) or Risk Assessment Report , BRD-clean | SysA
    ↓
Software Requirements Specification  ← BRD-clean
stage 02 -> FR, NFR | SysA
    ↓
Symantic Analasys <- FR, NFR, (As-Is HLD)
stage 03 -> High-Level Design(to-be){ Context Map,Aggregates} | Arh
    ↓
Invariant Analasys <- High-Level Design(to-be){ Context Map,Aggregates}
stage 04 ->  High-Level Design(to-be){ Context Map,Aggregates & Invariants, degradationStrategy},ADR | Arh
    ↓
Low Level Design <- High-Level Design(to-be){ Context Map,Aggregates}
stage 05 ->  LLD(to-be){ perContainer Contracts,perLayer Contracts, Happy/NegativePath, Event Maps}, | SysA
    ↓
stage 06 -> Task Plan{Tasks[], AcceptanceCriteria[]} (Jira Tasks linked to Updated ADR & Delta LLD)
            | SysA

### change request:

Formal <- change_request_input (Business, DevSecOps, TechDebt, Audit)
stage 00 -> CR-Brief (Change Request Brief) | SysA
    ↓
Impact Analysis <- Existing Artifacts (As-Is HLD, Current ADR, Current LLD, NFR Baseline)
stage 01 -> Impact Assessment Report (Blast Radius) & Routing Decision | SysA
    
#### Route A: Minor Change (e.g., UI, simple config)
stage 02 -> Delta SRS (FR_diff, NFR_diff) | SysA
    ↓
stage 05 -> Delta LLD (Updated specific Contracts, Event Maps, NegativePaths) | SysA
    ↓
stage 06 -> Task Plan{Tasks[], AcceptanceCriteria[]} (Jira Tasks with Acceptance Criteria) | SysA + TL
    
#### Route B: Major Change (e.g., new provider, security architecture flaw)
stage 02 -> Delta SRS (FR_diff, NFR_diff) | SysA
    ↓
Semantic Delta Analysis <- Delta SRS + Current HLD
stage 03 -> Delta HLD (Updated Context Map, Aggregates boundaries) | Arh
    ↓
Invariant Check <- Delta HLD
stage 04 -> Updated ADR (or new ADR superseding old), Updated Degradation Strategy | Arh
    ↓
Low Level Design Update <- Updated ADR + Delta HLD
stage 05 -> Delta LLD (perContainer Contracts diff, perLayer Contracts diff, Happy/NegativePath updates, Event Maps updates) | SysA
    ↓
stage 06 -> Task Plan{Tasks[], AcceptanceCriteria[]} (Jira Tasks linked to Updated ADR & Delta LLD) | SysA



### tdd

Test Contract ← perContainer Contracts, perLayer Contracts, Happy/NegativePath
stage 07 -> TestContractForm | QA
    ↓
Implementation Spec ← TestContractForm + LLD Contracts + ADR
stage 08 -> ImplementationSpecForm | dev
    ↓
QA Verdict ← TestContractForm + ImplementationResult
stage 09 -> verdict-form | QA
    ↓ (if fail)
Refactor Spec ← verdict-form.aspects + TestContractForm
stage 08.R -> RefactorSpecForm{ violations[], suggestedPatterns[], preserveTests: true }
              | DevOps → снова stage 09
