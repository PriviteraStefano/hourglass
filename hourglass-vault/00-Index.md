# Hourglass Documentation Index

Welcome to the Hourglass documentation hub! This is your central reference for understanding and developing the time entry and expense tracking system.

## 📚 Core Documentation

### Foundation
- [[01-System-Overview]] - What Hourglass is, why it exists, and how it works
- [[02-Architecture]] - Tech stack, system design, and component relationships
- [[03-Database-Schema]] - Complete data model and table references

### Development Guides
- **Backend Development**
  - [[04-Backend-Patterns]] - Handlers, API conventions, auth patterns
  - [[05-Auth-System]] - JWT, password hashing, token refresh
  - [[06-Middleware]] - Request processing pipeline

- **Frontend Development**
  - [[07-Frontend-Architecture]] - React, TanStack Router, component structure
  - [[08-API-Client]] - HTTP client, React Query patterns
  - [[09-State-Management]] - Query client configuration, data fetching

### Features
- [[10-Time-Entries]] - Time tracking, approval workflow
- [[11-Expenses]] - Expense management, role-based approvals
- [[12-Contracts-Projects]] - Contract and project management
- [[13-Organization-Users]] - Multi-tenant org model, user management
- [[14-CSV-Exports]] - Report generation with role-based scoping

### Operations
- [[15-Development-Setup]] - Local development environment
- [[16-Database-Migrations]] - Schema versioning and evolution
- [[17-Testing]] - Testing strategies and conventions
- [[18-Deployment]] - Docker, environment variables, production setup

## 🎯 Quick Navigation

| Need... | See... |
|---------|--------|
| Getting started | [[15-Development-Setup]], then [[02-Architecture]] |
| Adding a new API endpoint | [[04-Backend-Patterns]], [[05-Auth-System]] |
| Understanding data flow | [[02-Architecture]], [[03-Database-Schema]] |
| Building a React feature | [[07-Frontend-Architecture]], [[08-API-Client]] |
| Deploying to production | [[18-Deployment]] |
| Database structure | [[03-Database-Schema]], [[16-Database-Migrations]] |

## 📊 System Overview

**Hourglass** is a full-stack TypeScript/Go application for:
- Time entry tracking with per-project granularity
- Expense management with approval workflows
- Multi-organization support with role-based access
- Contract and project management with shared resources

**Key Technologies:**
- Backend: Go 1.26.1, PostgreSQL, standard library HTTP
- Frontend: React 19, TanStack Router, TanStack Query, Vite
- Deployment: Docker, PostgreSQL

## 🔑 Key Concepts

**Roles**: `employee`, `manager`, `finance`, `customer` — control approval workflows and data visibility

**Entry Status**: `draft` → `submitted` → `pending_manager` → `pending_finance` → `approved`/`rejected`

**Governance Models**: `creator_controlled`, `unanimous`, `majority` — define approval rules

**Multi-Tenancy**: Users belong to organizations; contracts/projects can be shared across orgs

## 📊 Complete Documentation Set

### Foundation (Start Here)
1. [[01-System-Overview]] - Business logic and workflows
2. [[02-Architecture]] - System design and tech stack
3. [[03-Database-Schema]] - Complete ERD reference
4. [[15-Development-Setup]] - Local environment setup

### Backend Development
5. [[04-Backend-Patterns]] - Handler, request/response patterns
6. [[05-Auth-System]] - JWT, password hashing, token management
7. [[06-Middleware]] - Request processing pipeline

### Frontend Development
8. [[07-Frontend-Architecture]] - React, TanStack Router, components
9. [[08-API-Client]] - HTTP client, React Query patterns

### Features (Functional Domain)
10. [[10-Time-Entries]] - Time tracking and approval workflow
11. [[11-Expenses]] - Expense management and categories
12. [[12-Contracts-Projects]] - Shared resources and governance
13. [[13-Organization-Users]] - Multi-tenancy and role-based access
14. [[14-CSV-Exports]] - Report generation with scoped data

### Operations & Deployment
15. [[16-Database-Migrations]] - Schema versioning and evolution
16. [[17-Testing]] - Backend and frontend testing strategies
17. [[18-Deployment]] - Docker, Kubernetes, infrastructure

## 📝 Last Updated
Generated: 2026-04-02

## Statistics
- **18 Documentation Files** covering full system
- **Architecture Diagrams** for system overview
- **API Endpoint Reference** for all features
- **Code Examples** in Go, TypeScript, and SQL
- **Deployment Guides** for production setup
- **Testing Patterns** for both backend and frontend

---

**New to the project?** Start with [[01-System-Overview]], then [[15-Development-Setup]].

**Contributing?** Review [[04-Backend-Patterns]] and [[07-Frontend-Architecture]] for conventions.

**Deploying?** Check [[18-Deployment]] and [[15-Development-Setup]] for infrastructure.
