-- +goose Up
-- +goose StatementBegin
-- Initial curated source list. Selection criteria:
--   tier1: wire services, peer-reviewed journals, .gov/.edu, dedicated fact-checkers
--   tier2: major newspapers of record with editorial standards
--   tier3: respected specialist outlets, broadcasters
-- Editorial: NOT exhaustive, NOT a value judgment on omitted outlets. Admin can
-- add/remove via /api/admin/sources. Vetting notes are short and citable.
INSERT INTO sources (domain, name, category, country, trust_tier, vetting_notes) VALUES
  -- ----- wire services -----
  ('reuters.com',       'Reuters',                'wire',        'GB', 'tier1', 'Trust Principles, third-party Trust Project verified.'),
  ('apnews.com',        'Associated Press',       'wire',        'US', 'tier1', 'Independent non-profit; rigorous corrections policy.'),
  ('afp.com',           'Agence France-Presse',   'wire',        'FR', 'tier1', 'Major international wire service.'),
  ('bloomberg.com',     'Bloomberg',              'wire',        'US', 'tier2', 'Strong on finance; mixed on lifestyle.'),
  ('dpa.com',           'Deutsche Presse-Agentur','wire',        'DE', 'tier1', 'German national wire service.'),

  -- ----- newspapers of record -----
  ('nytimes.com',       'The New York Times',     'newspaper',   'US', 'tier2', 'Public corrections page; editorial separation from opinion.'),
  ('washingtonpost.com','The Washington Post',    'newspaper',   'US', 'tier2', ''),
  ('wsj.com',           'The Wall Street Journal','newspaper',   'US', 'tier2', 'News desk; opinion section excluded for verdict purposes.'),
  ('bbc.com',           'BBC News',               'newspaper',   'GB', 'tier1', 'Public broadcaster; royal charter editorial standards.'),
  ('theguardian.com',   'The Guardian',           'newspaper',   'GB', 'tier2', 'Open corrections; Reader''s Editor.'),
  ('ft.com',            'Financial Times',        'newspaper',   'GB', 'tier2', ''),
  ('lemonde.fr',        'Le Monde',               'newspaper',   'FR', 'tier1', 'French newspaper of record.'),
  ('liberation.fr',     'Libération',             'newspaper',   'FR', 'tier2', ''),
  ('spiegel.de',        'Der Spiegel',            'newspaper',   'DE', 'tier2', 'German weekly with strong fact-check desk.'),
  ('sueddeutsche.de',   'Süddeutsche Zeitung',    'newspaper',   'DE', 'tier2', ''),
  ('zeit.de',           'Die Zeit',               'newspaper',   'DE', 'tier2', ''),
  ('nzz.ch',            'Neue Zürcher Zeitung',   'newspaper',   'CH', 'tier2', ''),
  ('elpais.com',        'El País',                'newspaper',   'ES', 'tier2', ''),
  ('corriere.it',       'Corriere della Sera',    'newspaper',   'IT', 'tier2', ''),
  ('repubblica.it',     'La Repubblica',          'newspaper',   'IT', 'tier2', ''),
  ('asahi.com',         'The Asahi Shimbun',      'newspaper',   'JP', 'tier2', ''),
  ('nhk.or.jp',         'NHK',                    'newspaper',   'JP', 'tier2', 'Public broadcaster.'),
  ('abc.net.au',        'ABC News (Australia)',   'newspaper',   'AU', 'tier1', 'Public broadcaster.'),
  ('cbc.ca',            'CBC News',               'newspaper',   'CA', 'tier1', 'Public broadcaster.'),
  ('theglobeandmail.com','The Globe and Mail',    'newspaper',   'CA', 'tier2', ''),
  ('haaretz.com',       'Haaretz',                'newspaper',   'IL', 'tier2', ''),
  ('thehindu.com',      'The Hindu',              'newspaper',   'IN', 'tier2', ''),
  ('indianexpress.com', 'The Indian Express',     'newspaper',   'IN', 'tier2', ''),
  ('scmp.com',          'South China Morning Post','newspaper',  'HK', 'tier2', ''),
  ('straitstimes.com',  'The Straits Times',      'newspaper',   'SG', 'tier2', ''),
  ('npr.org',           'NPR',                    'newspaper',   'US', 'tier2', 'Public radio; ethics handbook.'),
  ('pbs.org',           'PBS',                    'newspaper',   'US', 'tier2', ''),
  ('propublica.org',    'ProPublica',             'newspaper',   'US', 'tier1', 'Non-profit investigative; Pulitzer-recognized methodology.'),
  ('aljazeera.com',     'Al Jazeera English',     'newspaper',   'QA', 'tier2', ''),

  -- ----- magazines / longform with strong fact-check -----
  ('economist.com',     'The Economist',          'magazine',    'GB', 'tier2', ''),
  ('newyorker.com',     'The New Yorker',         'magazine',    'US', 'tier2', 'In-house fact-checking desk.'),
  ('atlanticmedia.com', 'The Atlantic',           'magazine',    'US', 'tier2', ''),
  ('nationalgeographic.com','National Geographic','magazine',    'US', 'tier2', ''),

  -- ----- dedicated fact-checkers -----
  ('snopes.com',        'Snopes',                 'factcheck',   'US', 'tier1', 'IFCN signatory.'),
  ('politifact.com',    'PolitiFact',             'factcheck',   'US', 'tier1', 'IFCN signatory.'),
  ('factcheck.org',     'FactCheck.org',          'factcheck',   'US', 'tier1', 'Annenberg Public Policy Center.'),
  ('apnews.com/hub/ap-fact-check','AP Fact Check','factcheck',   'US', 'tier1', 'Wire-service fact-check desk.'),
  ('reuters.com/fact-check','Reuters Fact Check', 'factcheck',   'GB', 'tier1', ''),
  ('fullfact.org',      'Full Fact',              'factcheck',   'GB', 'tier1', 'IFCN signatory.'),
  ('correctiv.org',     'Correctiv',              'factcheck',   'DE', 'tier1', 'German non-profit fact-checker.'),
  ('afp.com/en/afp-services/fact-check','AFP Fact Check','factcheck','FR', 'tier1', ''),
  ('maldita.es',        'Maldita.es',             'factcheck',   'ES', 'tier1', 'IFCN signatory.'),
  ('chequeado.com',     'Chequeado',              'factcheck',   'AR', 'tier1', 'IFCN signatory.'),
  ('boomlive.in',       'BOOM',                   'factcheck',   'IN', 'tier2', 'IFCN signatory.'),
  ('africacheck.org',   'Africa Check',           'factcheck',   'ZA', 'tier1', 'IFCN signatory.'),

  -- ----- academic + peer-review -----
  ('nature.com',        'Nature',                 'academic',    'GB', 'tier1', 'Peer-reviewed scientific journal.'),
  ('science.org',       'Science (AAAS)',         'academic',    'US', 'tier1', 'Peer-reviewed scientific journal.'),
  ('cell.com',          'Cell Press',             'academic',    'US', 'tier1', 'Peer-reviewed life science.'),
  ('thelancet.com',     'The Lancet',             'academic',    'GB', 'tier1', 'Peer-reviewed medical journal.'),
  ('nejm.org',          'New England Journal of Medicine','academic','US','tier1','Peer-reviewed medical journal.'),
  ('jamanetwork.com',   'JAMA Network',           'academic',    'US', 'tier1', ''),
  ('bmj.com',           'The BMJ',                'academic',    'GB', 'tier1', ''),
  ('plos.org',          'PLOS',                   'academic',    'US', 'tier2', 'Open-access peer-review.'),
  ('arxiv.org',         'arXiv',                  'academic',    'US', 'tier3', 'Preprints — NOT peer-reviewed; cite with caution.'),
  ('biorxiv.org',       'bioRxiv',                'academic',    'US', 'tier3', 'Preprints — NOT peer-reviewed.'),
  ('pubmed.ncbi.nlm.nih.gov','PubMed',            'academic',    'US', 'tier1', 'NIH-curated literature index.'),
  ('scholar.google.com','Google Scholar',         'academic',    'US', 'tier3', 'Index, not vetted; check upstream source.'),
  ('mpg.de',            'Max Planck Society',     'academic',    'DE', 'tier1', ''),

  -- ----- government / multilateral -----
  ('who.int',           'World Health Organization','government','CH', 'tier1', 'UN specialized agency.'),
  ('un.org',            'United Nations',         'government',  'US', 'tier1', ''),
  ('europa.eu',         'European Union',         'government',  'BE', 'tier1', 'Includes Eurostat, EMA, ECDC.'),
  ('cdc.gov',           'US CDC',                 'government',  'US', 'tier1', ''),
  ('nih.gov',           'US NIH',                 'government',  'US', 'tier1', ''),
  ('fda.gov',           'US FDA',                 'government',  'US', 'tier1', ''),
  ('nasa.gov',          'NASA',                   'government',  'US', 'tier1', ''),
  ('noaa.gov',          'NOAA',                   'government',  'US', 'tier1', ''),
  ('usgs.gov',          'US Geological Survey',   'government',  'US', 'tier1', ''),
  ('census.gov',        'US Census Bureau',       'government',  'US', 'tier1', ''),
  ('bls.gov',           'US Bureau of Labor Statistics','government','US','tier1',''),
  ('sec.gov',           'US SEC',                 'government',  'US', 'tier1', ''),
  ('ipcc.ch',           'IPCC',                   'government',  'CH', 'tier1', 'Intergovernmental Panel on Climate Change.'),
  ('iea.org',           'International Energy Agency','government','FR','tier1',''),
  ('imf.org',           'International Monetary Fund','government','US','tier1',''),
  ('worldbank.org',     'World Bank',             'government',  'US', 'tier1', ''),
  ('rki.de',            'Robert Koch Institut',   'government',  'DE', 'tier1', 'German public health institute.'),
  ('gov.uk',            'UK Government',          'government',  'GB', 'tier1', ''),
  ('parliament.uk',     'UK Parliament',          'government',  'GB', 'tier1', ''),
  ('ons.gov.uk',        'UK Office for National Statistics','government','GB','tier1',''),
  ('canada.ca',         'Government of Canada',   'government',  'CA', 'tier1', ''),
  ('bundesregierung.de','German Federal Government','government','DE', 'tier1', ''),
  ('admin.ch',          'Swiss Federal Authorities','government','CH', 'tier1', ''),
  ('ecdc.europa.eu',    'European CDC',           'government',  'SE', 'tier1', ''),

  -- ----- encyclopedia / reference -----
  ('en.wikipedia.org',  'Wikipedia (EN)',         'encyclopedia','US', 'tier3', 'Useful for orientation; verify citations independently.'),
  ('de.wikipedia.org',  'Wikipedia (DE)',         'encyclopedia','DE', 'tier3', ''),
  ('britannica.com',    'Encyclopaedia Britannica','encyclopedia','US','tier2', ''),
  ('plato.stanford.edu','Stanford Encyclopedia of Philosophy','encyclopedia','US','tier1','Peer-reviewed philosophy reference.'),

  -- ----- specialist -----
  ('cjr.org',           'Columbia Journalism Review','specialist','US','tier2',''),
  ('niemanreports.org', 'Nieman Reports',         'specialist',  'US', 'tier2', ''),
  ('poynter.org',       'Poynter',                'specialist',  'US', 'tier2', ''),
  ('arstechnica.com',   'Ars Technica',           'specialist',  'US', 'tier2', 'Strong on tech/science.'),
  ('technologyreview.com','MIT Technology Review','specialist',  'US', 'tier2', ''),
  ('quantamagazine.org','Quanta Magazine',        'specialist',  'US', 'tier2', 'Simons Foundation; rigorous science coverage.'),
  ('chemistryworld.com','Chemistry World',        'specialist',  'GB', 'tier2', ''),
  ('newscientist.com',  'New Scientist',          'specialist',  'GB', 'tier2', ''),
  ('skepticalinquirer.org','Skeptical Inquirer',  'specialist',  'US', 'tier2', ''),
  ('skepticalscience.com','Skeptical Science',    'specialist',  'AU', 'tier2', 'Climate science explainer.'),

  -- ----- primary corporate / org sources (low default tier; useful as primary but biased) -----
  ('blogs.microsoft.com','Microsoft (official blog)','primary',  'US', 'tier3', 'Primary source for the company''s own claims.'),
  ('googleblog.com',    'Google (official blog)', 'primary',     'US', 'tier3', ''),
  ('blog.openai.com',   'OpenAI (official blog)', 'primary',     'US', 'tier3', '')
ON CONFLICT (domain) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- Seed removal is selective so admin-added rows don't get clobbered.
DELETE FROM sources WHERE added_by IS NULL;
