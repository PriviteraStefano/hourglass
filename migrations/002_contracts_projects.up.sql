-- Enable UUID extension (already done in 001, but idempotent)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Contracts table (shared resources)
CREATE TABLE contracts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    km_rate NUMERIC(10, 2) DEFAULT 0.00,
    currency VARCHAR(3) DEFAULT 'EUR',
    governance_model VARCHAR(50) NOT NULL CHECK (governance_model IN ('creator_controlled', 'unanimous', 'majority')),
    created_by_org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    is_shared BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Projects table (shared resources)
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('billable', 'internal')),
    contract_id UUID REFERENCES contracts(id) ON DELETE CASCADE,
    governance_model VARCHAR(50) NOT NULL CHECK (governance_model IN ('creator_controlled', 'unanimous', 'majority')),
    created_by_org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    is_shared BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Contract adoptions (organizations adopting shared contracts)
CREATE TABLE contract_adoptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id UUID REFERENCES contracts(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    adopted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(contract_id, organization_id)
);

-- Project adoptions (organizations adopting shared projects)
CREATE TABLE project_adoptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    adopted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(project_id, organization_id)
);

-- Indexes for performance
CREATE INDEX idx_contracts_created_by_org ON contracts(created_by_org_id);
CREATE INDEX idx_contracts_is_shared ON contracts(is_shared) WHERE is_shared = TRUE;
CREATE INDEX idx_projects_contract_id ON projects(contract_id);
CREATE INDEX idx_projects_created_by_org ON projects(created_by_org_id);
CREATE INDEX idx_projects_is_shared ON projects(is_shared) WHERE is_shared = TRUE;
CREATE INDEX idx_contract_adoptions_contract_id ON contract_adoptions(contract_id);
CREATE INDEX idx_contract_adoptions_organization_id ON contract_adoptions(organization_id);
CREATE INDEX idx_project_adoptions_project_id ON project_adoptions(project_id);
CREATE INDEX idx_project_adoptions_organization_id ON project_adoptions(organization_id);