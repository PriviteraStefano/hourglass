# Documentation Structure Overview

## Complete Vault Map

```mermaid
graph TB
    Index[00-Index.md<br/>Central Hub]
    
    subgraph Features["📖 01-Features (User-Facing)"]
        F_Template[TEMPLATE.md]
        F04[F04: User Authentication]
        F05[F05: Org Bootstrap]
        F06[F06: Invitation System]
    end
    
    subgraph Technical["🔧 02-Technical (Implementation)"]
        T_Template[TEMPLATE.md]
        T01[T01: Hexagonal Architecture]
        T02[T02: Auth Implementation]
    end
    
    subgraph Schema["🏗️ 03-Schema (Design)"]
        S_Template[TEMPLATE.md]
        S01[S01: Database ERD]
        S02[S02: Domain Models]
        S03[S03: Ports & Interfaces]
        S04[S04: API Contracts]
        S05[S05: State Machines]
    end
    
    subgraph Legacy["📚 LEGACY (Previous Docs)"]
        L01[01-18: Original docs]
    end
    
    Index --> Features
    Index --> Technical
    Index --> Schema
    Index --> Legacy
    
    Features -.->|Uses Templates| F_Template
    Technical -.->|Uses Templates| T_Template
    Schema -.->|Uses Templates| S_Template
    
    F04 --> T02
    T02 --> S01
    T02 --> S02
    T02 --> S03
    T02 --> S04
    T02 --> S05
    
    style Index fill:#f9f,stroke:#333,stroke-width:2px
    style Features fill:#e1f5ff,stroke:#333
    style Technical fill:#fff4e1,stroke:#333
    style Schema fill:#e8f5e9,stroke:#333
    style Legacy fill:#f5f5f5,stroke:#999
```

## Document Relationships

### Authentication System (Complete)
```mermaid
flowchart LR
    F04[F04: User Stories] --> T02[T02: Implementation]
    T02 --> S01[S01: Database]
    T02 --> S02[S02: Domain Models]
    T02 --> S03[S03: Ports]
    T02 --> S04[S04: API Contracts]
    T02 --> S05[S05: State Machines]
    
    style F04 fill:#bbf,stroke:#333
    style T02 fill:#fbf,stroke:#333
    style S01 fill:#bfb,stroke:#333
    style S02 fill:#bfb,stroke:#333
    style S03 fill:#bfb,stroke:#333
    style S04 fill:#bfb,stroke:#333
    style S05 fill:#bfb,stroke:#333
```

### Next Features to Document
```mermaid
graph LR
    Auth[✅ Auth System] --> TimeEntries[⏳ Time Entries]
    TimeEntries --> Expenses[⏳ Expenses]
    Expenses --> Contracts[⏳ Contracts]
    Contracts --> Projects[⏳ Projects]
    
    style Auth fill:#9f9,stroke:#333
    style TimeEntries fill:#ff9,stroke:#333
    style Expenses fill:#ff9,stroke:#333
    style Contracts fill:#ff9,stroke:#333
    style Projects fill:#ff9,stroke:#333
```

## File Naming Convention

### Features
- Format: `F##-Feature-Name.md`
- Example: `F04-User-Authentication.md`
- Numbered for Obsidian sorting

### Technical
- Format: `T##-Topic-Name.md`
- Example: `T01-Hexagonal-Architecture.md`
- Numbered for Obsidian sorting

### Schema
- Format: `S##-Topic-Name.md`
- Example: `S01-Database-ERD.md`
- Numbered for Obsidian sorting

## Cross-Reference Pattern

All documents use wiki-style links:
- `[[F04-User-Authentication]]` - Link to feature
- `[[T02-Auth-Implementation]]` - Link to technical
- `[[S01-Database-ERD]]` - Link to schema

Mermaid diagrams show relationships visually.

## Automation Integration

```mermaid
flowchart TD
    PR[GitHub PR Merged] --> Script[generate-docs-draft.sh]
    Script --> Draft[Create Draft Doc]
    Draft --> Complete[Developer Completes Checklists]
    Complete --> Move[Move to Appropriate Folder]
    Move --> Check[docs-check.sh]
    Check --> CI[GitHub Actions]
    CI --> Report[Status Report]
```

## Quick Reference

| Need | Go To |
|------|-------|
| User workflow | `01-Features/FXX-*.md` |
| How to implement | `02-Technical/TXX-*.md` |
| Database structure | `03-Schema/S01-Database-ERD.md` |
| API spec | `03-Schema/S04-API-Contracts.md` |
| State transitions | `03-Schema/S05-State-Machines.md` |
| Architecture pattern | `02-Technical/T01-Hexagonal-Architecture.md` |

