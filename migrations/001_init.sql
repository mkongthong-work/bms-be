-- ============================================================================
-- BMS Admin — โครงสร้างฐานข้อมูล PostgreSQL 15+
-- ครอบคลุม: ระบบกลาง · แคตตาล็อกสินค้า · เอกสารงานขาย · พนักงาน/เงินเดือน
-- หลักการ: เก็บเงินเป็น NUMERIC(14,2) · snapshot ข้อมูลลง เอกสาร ณ วันออก
--          soft delete เฉพาะที่จำเป็น · เลขรันเอกสารกันชนด้วย row lock
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";   -- email case-insensitive

-- ============================== ระบบกลาง ==================================

CREATE TABLE companies (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name_th         text NOT NULL,
    name_en         text NOT NULL DEFAULT '',
    address_lines   text[] NOT NULL DEFAULT '{}',
    tax_id          text NOT NULL DEFAULT '',
    phone           text NOT NULL DEFAULT '',
    email           text NOT NULL DEFAULT '',
    vat_registered  boolean NOT NULL DEFAULT false,  -- สวิตช์ VAT ทั้งระบบ
    logo_png        bytea,                            -- โลโก้ฝังลง PDF
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code        text NOT NULL UNIQUE,                 -- ADMIN / SALES / ACCOUNT / HR
    name_th     text NOT NULL,
    permissions jsonb NOT NULL DEFAULT '{}'           -- {"documents":"rw","payroll":"none",...}
);

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    uuid NOT NULL REFERENCES companies(id),
    role_id       uuid NOT NULL REFERENCES roles(id),
    email         citext NOT NULL UNIQUE,
    password_hash text NOT NULL,                      -- bcrypt cost >= 12
    display_name  text NOT NULL,
    is_active     boolean NOT NULL DEFAULT true,
    last_login_at timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- เลขรันเอกสาร แยกตาม ประเภท × ปี พ.ศ. — จองเลขใน transaction เดียวกับ INSERT
CREATE TABLE document_sequences (
    company_id  uuid NOT NULL REFERENCES companies(id),
    doc_type    text NOT NULL CHECK (doc_type IN ('QT','INV','RC','DO','CN')),
    year_be     int  NOT NULL,                        -- 2569
    last_number int  NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, doc_type, year_be)
);

-- ฟังก์ชันจองเลขเอกสารแบบ atomic (SELECT ... FOR UPDATE โดยนัยผ่าน UPDATE)
-- คืนค่า เช่น 'QT-2569/0129' — เรียกใน transaction เดียวกับการ INSERT documents
CREATE OR REPLACE FUNCTION next_document_number(p_company uuid, p_type text, p_year int)
RETURNS text LANGUAGE plpgsql AS $$
DECLARE n int;
BEGIN
    INSERT INTO document_sequences (company_id, doc_type, year_be, last_number)
    VALUES (p_company, p_type, p_year, 1)
    ON CONFLICT (company_id, doc_type, year_be)
    DO UPDATE SET last_number = document_sequences.last_number + 1
    RETURNING last_number INTO n;
    RETURN format('%s-%s/%s', p_type, p_year, lpad(n::text, 4, '0'));
END $$;

-- ============================== แคตตาล็อก ==================================

CREATE TABLE categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name_th    text NOT NULL,
    name_en    text NOT NULL DEFAULT '',
    slug       text NOT NULL UNIQUE,                  -- ใช้กับเว็บหน้าบ้าน
    sort_order int  NOT NULL DEFAULT 0
);

CREATE TABLE products (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id uuid REFERENCES categories(id),
    sku         text NOT NULL UNIQUE,                 -- HPT-2500
    name_th     text NOT NULL,
    name_en     text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    unit        text NOT NULL DEFAULT 'ชิ้น',
    sell_price  numeric(14,2) NOT NULL DEFAULT 0 CHECK (sell_price >= 0),
    cost_price  numeric(14,2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
    status      text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published')),
    deleted_at  timestamptz,                          -- soft delete: เอกสารเก่ายังอ้างได้
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_products_search ON products USING gin (to_tsvector('simple', sku || ' ' || name_th || ' ' || name_en));
CREATE INDEX idx_products_category ON products (category_id) WHERE deleted_at IS NULL;

CREATE TABLE product_images (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    file_path  text NOT NULL,                         -- object storage key (ห้ามเก็บ URL สาธารณะตรง ๆ)
    sort_order int  NOT NULL DEFAULT 0
);

CREATE TABLE product_specs (                          -- สเปค key-value ตามฟอร์มสินค้า
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id uuid NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    label      text NOT NULL,                         -- น้ำหนักยกสูงสุด
    value      text NOT NULL,                         -- 2,500 กก.
    sort_order int  NOT NULL DEFAULT 0
);

-- ============================== ลูกค้า ====================================

CREATE TABLE customers (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name         text NOT NULL,
    address_lines text[] NOT NULL DEFAULT '{}',
    tax_id       text NOT NULL DEFAULT '',
    branch       text NOT NULL DEFAULT 'สำนักงานใหญ่', -- จำเป็นบนใบกำกับภาษี
    contact_name text NOT NULL DEFAULT '',
    phone        text NOT NULL DEFAULT '',
    email        text NOT NULL DEFAULT '',
    note         text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_customers_name ON customers USING gin (to_tsvector('simple', name));

-- =========================== เอกสารงานขาย =================================
-- ตารางเดียวทุกประเภท (QT/INV/RC/DO/CN) — แปลงเอกสารต่อกันผ่าน source_document_id
-- ข้อมูลลูกค้า/VAT เก็บเป็น snapshot: แก้ทะเบียนภายหลังไม่กระทบเอกสารที่ออกแล้ว

CREATE TABLE documents (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id         uuid NOT NULL REFERENCES companies(id),
    doc_type           text NOT NULL CHECK (doc_type IN ('QT','INV','RC','DO','CN')),
    doc_number         text NOT NULL UNIQUE,          -- จาก next_document_number()
    customer_id        uuid NOT NULL REFERENCES customers(id),
    source_document_id uuid REFERENCES documents(id), -- QT ต้นทางของ INV ฯลฯ
    issue_date         date NOT NULL DEFAULT CURRENT_DATE,
    due_date           date,                          -- ยืนราคาถึง / ครบกำหนดชำระ
    status             text NOT NULL DEFAULT 'draft'
                       CHECK (status IN ('draft','sent','approved','partial','paid','void')),
    bill_discount      numeric(14,2) NOT NULL DEFAULT 0 CHECK (bill_discount >= 0),
    withholding_pct    numeric(5,2)  NOT NULL DEFAULT 0 CHECK (withholding_pct BETWEEN 0 AND 100),
    vat_applied        boolean NOT NULL,              -- snapshot สวิตช์ VAT ณ วันออก
    customer_snapshot  jsonb NOT NULL,                -- {name,address_lines,tax_id,branch,contact}
    salesperson        text NOT NULL DEFAULT '',
    payment_terms      text NOT NULL DEFAULT '',
    notes              text[] NOT NULL DEFAULT '{}',
    subtotal           numeric(14,2) NOT NULL DEFAULT 0,  -- denormalized เพื่อ report เร็ว
    vat_amount         numeric(14,2) NOT NULL DEFAULT 0,
    grand_total        numeric(14,2) NOT NULL DEFAULT 0,
    created_by         uuid NOT NULL REFERENCES users(id),
    voided_reason      text,                          -- ยกเลิกต้องมีเหตุผล (audit)
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_documents_list   ON documents (doc_type, status, issue_date DESC);
CREATE INDEX idx_documents_cust   ON documents (customer_id, issue_date DESC);
CREATE INDEX idx_documents_source ON documents (source_document_id);

CREATE TABLE document_items (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    product_id  uuid REFERENCES products(id),         -- NULL = รายการอิสระ (ค่าบริการพิเศษ)
    sku         text NOT NULL DEFAULT '',
    name        text NOT NULL,                        -- snapshot ชื่อ ณ วันออก
    detail      text NOT NULL DEFAULT '',
    qty         numeric(12,3) NOT NULL CHECK (qty >= 0),
    unit        text NOT NULL DEFAULT 'ชิ้น',
    unit_price  numeric(14,2) NOT NULL CHECK (unit_price >= 0),
    discount    numeric(14,2) NOT NULL DEFAULT 0 CHECK (discount >= 0),
    sort_order  int NOT NULL DEFAULT 0
);
CREATE INDEX idx_document_items_doc ON document_items (document_id, sort_order);

CREATE TABLE payments (                               -- รับชำระ (รองรับผ่อนหลายงวด)
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id uuid NOT NULL REFERENCES documents(id),
    paid_date   date NOT NULL DEFAULT CURRENT_DATE,
    amount      numeric(14,2) NOT NULL CHECK (amount > 0),
    method      text NOT NULL CHECK (method IN ('cash','transfer','cheque','card')),
    bank_ref    text NOT NULL DEFAULT '',
    recorded_by uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- ========================= พนักงาน + เงินเดือน =============================

CREATE TABLE employees (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code         text NOT NULL UNIQUE,                -- EMP-001
    full_name    text NOT NULL,
    position     text NOT NULL DEFAULT '',
    wage_type    text NOT NULL CHECK (wage_type IN ('daily','monthly')),
    wage_rate    numeric(12,2) NOT NULL CHECK (wage_rate >= 0), -- บาท/วัน หรือ บาท/เดือน
    ot_rate_mult numeric(4,2) NOT NULL DEFAULT 1.5,   -- ตัวคูณ OT
    national_id  text NOT NULL DEFAULT '',            -- แสดงแบบ mask ใน UI, จำกัดสิทธิ์ HR
    sso_number   text NOT NULL DEFAULT '',
    hired_date   date,
    bank_account text NOT NULL DEFAULT '',            -- จำกัดสิทธิ์ HR/ACCOUNT เท่านั้น
    is_active    boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE attendances (                            -- กรอกเป็น grid รายเดือน
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id uuid NOT NULL REFERENCES employees(id),
    work_date   date NOT NULL,
    status      text NOT NULL DEFAULT 'work'
                CHECK (status IN ('work','half','leave','sick','absent','holiday')),
    ot_hours    numeric(4,1) NOT NULL DEFAULT 0 CHECK (ot_hours >= 0),
    note        text NOT NULL DEFAULT '',
    UNIQUE (employee_id, work_date)
);
CREATE INDEX idx_attendances_month ON attendances (employee_id, work_date);

CREATE TABLE payroll_runs (                           -- งวดจ่ายเงินเดือน
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    period_start date NOT NULL,
    period_end   date NOT NULL CHECK (period_end >= period_start),
    status       text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','approved','paid')),
    approved_by  uuid REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period_start, period_end)
);

CREATE TABLE payroll_items (                          -- หนึ่งแถว = สลิปหนึ่งคนต่อหนึ่งงวด
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_run_id uuid NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
    employee_id    uuid NOT NULL REFERENCES employees(id),
    work_days      numeric(5,1)  NOT NULL DEFAULT 0,  -- snapshot จาก attendances
    ot_hours       numeric(6,1)  NOT NULL DEFAULT 0,
    base_pay       numeric(12,2) NOT NULL DEFAULT 0,
    ot_pay         numeric(12,2) NOT NULL DEFAULT 0,
    other_income   numeric(12,2) NOT NULL DEFAULT 0,  -- เบี้ยขยัน/ค่าเดินทาง
    deductions     numeric(12,2) NOT NULL DEFAULT 0,  -- เบิกล่วงหน้า/หักอื่น
    sso_amount     numeric(12,2) NOT NULL DEFAULT 0,  -- 5% เพดาน 750 (คำนวณฝั่ง Go)
    net_pay        numeric(12,2) NOT NULL DEFAULT 0,
    detail         jsonb NOT NULL DEFAULT '{}',       -- breakdown แสดงบนสลิป PDF
    UNIQUE (payroll_run_id, employee_id)
);

-- ============================== Audit log ==================================
-- บันทึกการเปลี่ยนแปลงสำคัญ (ออก/ยกเลิกเอกสาร, อนุมัติเงินเดือน)

CREATE TABLE audit_logs (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    uuid REFERENCES users(id),
    action     text NOT NULL,                         -- document.void, payroll.approve, ...
    entity     text NOT NULL,                         -- documents / payroll_runs / ...
    entity_id  uuid NOT NULL,
    payload    jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_entity ON audit_logs (entity, entity_id, created_at DESC);

-- ============================ ข้อมูลตั้งต้น ================================

INSERT INTO roles (code, name_th, permissions) VALUES
 ('ADMIN',   'ผู้ดูแลระบบ',  '{"*":"rw"}'),
 ('SALES',   'ฝ่ายขาย',     '{"documents":"rw","products":"rw","customers":"rw","payroll":"none","employees":"none"}'),
 ('ACCOUNT', 'บัญชี',       '{"documents":"rw","payments":"rw","reports":"r","payroll":"r"}'),
 ('HR',      'ฝ่ายบุคคล',   '{"employees":"rw","attendances":"rw","payroll":"rw","documents":"none"}');
